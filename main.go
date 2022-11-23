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

	flag.BoolVar(&appFlags.UseMqtt, "mqtt-activated", false, "Use MQTT server instead of reader for input")
	flag.StringVar(&appFlags.MqttServer, "mqtt-server", "", "MQTT Server address")
	flag.StringVar(&appFlags.MqttUser, "mqtt-user", "", "MQTT User name")
	flag.StringVar(&appFlags.MqttPassword, "mqtt-password", "", "MQTT Password")
	flag.IntVar(&appFlags.MqttPort, "mqtt-port", 1883, "MQTT Port")
	flag.StringVar(&appFlags.MqttTopic, "mqtt-topic", "nfcuid/reader", "MQTT Topic")
	flag.StringVar(&appFlags.MqttId, "mqtt-id", "nfcuidclient", "MQTT id")
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
