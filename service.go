package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/ebfe/scard"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/taglme/string2keyboard"
)

type Service interface {
	Start()
	Flags() Flags
}

func NewService(flags Flags) Service {
	return &service{flags}
}

type Flags struct {
	CapsLock     bool
	Reverse      bool
	Decimal      bool
	EndChar      CharFlag
	InChar       CharFlag
	Device       int
	GetAsDecimal bool
	UseMqtt      bool
	MqttServer   string
	MqttUser     string
	MqttPassword string
	MqttPort     int
	MqttTopic    string
	MqttId       string
	Debug        bool
}

type service struct {
	flags Flags
}

func (s *service) Start() {

	if s.flags.UseMqtt {
		var bCleanSession bool = false
		var qos int = 0

		opts := MQTT.NewClientOptions()
		var broker string = fmt.Sprintf(s.flags.MqttServer+":%d", s.flags.MqttPort)
		if !strings.HasPrefix(broker, "tcp://") {
			broker = "tcp://" + broker
		}
		fmt.Println("Using MQTT")
		//tcp://iot.eclipse.org:1883
		//fmt.Printf("\tserver: :    	%s\n", *&s.flags.MqttServer)
		fmt.Printf("\tbroker: :    	%s\n", broker)
		if s.flags.Debug {
			fmt.Printf("\tuser:			%s\n", s.flags.MqttUser)
			fmt.Printf("\tport:			%d\n", s.flags.MqttPort)
		}
		fmt.Printf("\ttopic:		%s\n", s.flags.MqttTopic)

		opts.AddBroker(broker)
		opts.SetClientID(s.flags.MqttId)
		opts.SetUsername(s.flags.MqttUser)
		opts.SetPassword(s.flags.MqttPassword)
		opts.SetCleanSession(bCleanSession)

		choke := make(chan [2]string)

		opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
			choke <- [2]string{msg.Topic(), string(msg.Payload())}
		})

		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		if token := client.Subscribe(s.flags.MqttTopic, byte(qos), nil); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			client.Disconnect(250)
			fmt.Println("Disconnected from server")
			os.Exit(1)
		}()

		//for receiveCount < *num {
		for {
			incoming := <-choke
			//fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
			if len(incoming[1]) > 0 {
				var inBytes []byte = getBytesFromString(incoming[1])
				if s.flags.Debug {
					fmt.Printf("GOT: %x\n", inBytes)
				}
				var result string = s.formatOutput(inBytes)
				if s.flags.Debug {
					fmt.Printf("OUTPUT: %s\n", result)
				} else {
					var err = string2keyboard.KeyboardWrite(result)
					if err != nil {
						fmt.Printf("Could write as keyboard output. Error: %s\n", err.Error())
					} else if s.flags.Debug {
						fmt.Printf("WROTE: %s\n", result)
					}

				}
			}
		}
	}
	//Establish a context
	ctx, err := scard.EstablishContext()
	if err != nil {
		errorExit(err)
	}
	defer ctx.Release()

	//List available readers
	readers, err := ctx.ListReaders()
	if err != nil {
		errorExit(err)
	}

	if len(readers) < 1 {
		errorExit(errors.New("Devices not found. Try to plug-in new device and restart"))
	}

	fmt.Printf("Found %d device:\n", len(readers))
	for i, reader := range readers {
		fmt.Printf("[%d] %s\n", i+1, reader)
	}

	if s.flags.Device == 0 {
		//Device should be selected by user input
		for {
			fmt.Print("Enter device number to start: ")
			inputReader := bufio.NewReader(os.Stdin)
			deviceStr, _ := inputReader.ReadString('\n')

			if runtime.GOOS == "windows" {
				deviceStr = strings.Replace(deviceStr, "\r\n", "", -1)
			} else {
				deviceStr = strings.Replace(deviceStr, "\n", "", -1)
			}
			deviceInt, err := strconv.Atoi(deviceStr)
			if err != nil {
				fmt.Println("Please input integer value")
				continue
			}
			if deviceInt < 0 {
				fmt.Println("Please input positive integer value")
				continue
			}
			if deviceInt > len(readers) {
				fmt.Printf("Value should be less than or equal to %d\n", len(readers))
				continue
			}
			s.flags.Device = deviceInt
			break
		}
	} else if s.flags.Device < 0 {
		errorExit(errors.New("Device flag should positive integer"))
		return
	} else if s.flags.Device > len(readers) {
		errorExit(errors.New("Device flag should not exceed the number of available devices"))
		return
	}

	fmt.Println("Selected device:")
	fmt.Printf("[%d] %s\n", s.flags.Device, readers[s.flags.Device-1])
	selectedReaders := []string{readers[s.flags.Device-1]}

	for {
		fmt.Println("Waiting for a Card")
		index, err := waitUntilCardPresent(ctx, selectedReaders)
		if err != nil {
			errorExit(err)
		}

		//Connect to card
		fmt.Println("Connecting to card...")
		card, err := ctx.Connect(selectedReaders[index], scard.ShareShared, scard.ProtocolAny)
		if err != nil {
			errorExit(err)
		}
		defer card.Disconnect(scard.ResetCard)

		//GET DATA command
		var cmd = []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}

		rsp, err := card.Transmit(cmd)
		if err != nil {
			errorExit(err)
		}

		if len(rsp) < 2 {
			fmt.Println("Not enough bytes in answer. Try again")
			card.Disconnect(scard.ResetCard)
			continue
		}

		//Check response code - two last bytes of response
		rspCodeBytes := rsp[len(rsp)-2 : len(rsp)]
		successResponseCode := []byte{0x90, 0x00}
		if !bytes.Equal(rspCodeBytes, successResponseCode) {
			fmt.Printf("Operation failed to complete. Error code % x\n", rspCodeBytes)
			card.Disconnect(scard.ResetCard)
			continue
		}

		uidBytes := rsp[0 : len(rsp)-2]
		fmt.Printf("UID is: % x\n", uidBytes)
		fmt.Printf("Writting as keyboard input...")
		err = string2keyboard.KeyboardWrite(s.formatOutput(uidBytes))
		if err != nil {
			fmt.Printf("Could write as keyboard output. Error: %s\n", err.Error())
		} else {
			fmt.Printf("Success!\n")
		}

		card.Disconnect(scard.ResetCard)

		//Wait while card will be released
		fmt.Print("Waiting for card release...")
		err = waitUntilCardRelease(ctx, selectedReaders, index)
		fmt.Println("Card released")

	}

}

func (s *service) Flags() Flags {
	return s.flags
}

func errorExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func (s *service) formatOutput(rx []byte) string {

	if s.flags.GetAsDecimal {
		return getDecFromHexArray(rx) + s.flags.EndChar.Output()
	}

	var output string
	//Reverse UID in flag set
	if s.flags.Reverse {
		for i, j := 0, len(rx)-1; i < j; i, j = i+1, j-1 {
			rx[i], rx[j] = rx[j], rx[i]
		}
	}

	for i, rxByte := range rx {
		var byteStr string
		if s.flags.Decimal {
			byteStr = fmt.Sprintf("%03d", rxByte)
		} else {
			if s.flags.CapsLock {
				byteStr = fmt.Sprintf("%02X", rxByte)
			} else {
				byteStr = fmt.Sprintf("%02x", rxByte)

			}

		}
		output = output + byteStr
		if i < len(rx)-1 {
			output = output + s.flags.InChar.Output()
		}

	}

	output = output + s.flags.EndChar.Output()
	return output
}
func getBytesFromString(inValue string) []byte {
	inValue = strings.Trim(inValue, " ")
	splitString := strings.Split(inValue, ":")
	var output []byte
	for _, s := range splitString {
		h, _ := hex.DecodeString(s)
		output = append(output, h...)
	}
	return output
}
func getDecFromHexArray(byteArr []byte) string {
	var sByte string
	for _, b := range byteArr {
		var s string = fmt.Sprintf("%x", b)
		sByte = sByte + s
	}
	if len(sByte) == 0 {
		return ""
	}
	dNum, err := strconv.ParseInt(sByte, 16, 64)
	if err != nil {
		errorExit(err)
	}
	var ret string = fmt.Sprintf("%010d", dNum)
	return ret
}

func waitUntilCardPresent(ctx *scard.Context, readers []string) (int, error) {
	rs := make([]scard.ReaderState, len(readers))
	for i := range rs {
		rs[i].Reader = readers[i]
		rs[i].CurrentState = scard.StateUnaware
	}

	for {
		for i := range rs {
			if rs[i].EventState&scard.StatePresent != 0 {
				return i, nil
			}
			rs[i].CurrentState = rs[i].EventState
		}
		err := ctx.GetStatusChange(rs, -1)
		if err != nil {
			return -1, err
		}
	}
}

func waitUntilCardRelease(ctx *scard.Context, readers []string, index int) error {
	rs := make([]scard.ReaderState, 1)

	rs[0].Reader = readers[index]
	rs[0].CurrentState = scard.StatePresent

	for {

		if rs[0].EventState&scard.StateEmpty != 0 {
			return nil
		}
		rs[0].CurrentState = rs[0].EventState

		err := ctx.GetStatusChange(rs, -1)
		if err != nil {
			return err
		}
	}
}
