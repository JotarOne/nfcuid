package main

import (
	"errors"
	"flag"
)

func main() {
	var appFlags Flags
	var endChar, inChar string
	var ok bool
	//Read application flags
	flag.StringVar(&endChar, "end-char", "none", "Character at the end of UID. Options: "+CharFlagOptions())
	flag.StringVar(&inChar, "in-char", "none", "Сharacter between bytes of UID. Options: "+CharFlagOptions())
	flag.BoolVar(&appFlags.CapsLock, "caps-lock", false, "UID with Caps Lock")
	flag.BoolVar(&appFlags.Reverse, "reverse", false, "UID reverse order")
	flag.BoolVar(&appFlags.Decimal, "decimal", false, "UID in decimal format")
	flag.IntVar(&appFlags.Device, "device", 0, "Device number to use")
	flag.BoolVar(&appFlags.GetAsDecimal, "getasdecimal", false, "Get decimal value of input")

	flag.BoolVar(&appFlags.UseMqttAsInput, "mqttInput", false, "Use MQTT server")
	flag.BoolVar(&appFlags.UseMqttAsOutput, "mqttOutput", false, "Use MQTT server")
	flag.StringVar(&appFlags.MqttServer, "mqttServer", "", "MQTT Server address")
	flag.StringVar(&appFlags.MqttUser, "mqttUser", "", "MQTT User name")
	flag.StringVar(&appFlags.MqttPassword, "mqttPassword", "", "MQTT Password")
	flag.IntVar(&appFlags.MqttPort, "mqttPort", 1883, "MQTT Port")

	flag.StringVar(&appFlags.MqttTopicIn, "mqttTopicRead", "nfcuid/reader-in", "MQTT Topic")
	flag.StringVar(&appFlags.MqttTopicOut, "mqttTopicWrite", "nfcuid/reader-out", "MQTT Topic")

	flag.StringVar(&appFlags.MqttId, "mqttId", "nfcuidclient", "MQTT id")

	flag.BoolVar(&appFlags.UseTcpSocket, "tcpSocketOutput", false, "Use tcp socket")
	flag.IntVar(&appFlags.TcpSocketPort, "tcpSocketPort", 10340, "Tcp socket port")
	flag.StringVar(&appFlags.TcpSocketAddress, "tcpSocketAddress", "127.0.0.1", "Tcp socket address")

	flag.BoolVar(&appFlags.Debug, "debug", false, "Output debug information")
	flag.Parse()

	//Check flags
	appFlags.EndChar, ok = StringToCharFlag(endChar)
	if !ok {
		errorExit(errors.New("Unknown end character flag. Run with '-h' flag to check options"))
		return
	}
	appFlags.InChar, ok = StringToCharFlag(inChar)
	if !ok {
		errorExit(errors.New("Unknown in character flag. Run with '-h' flag to check options"))
		return
	}

	service := NewService(appFlags)
	service.Start()

}
