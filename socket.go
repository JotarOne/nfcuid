package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func sendSocketMessage(address string, port int, message string) {
	CONNECT := address + ":" + strconv.Itoa(port)
	var writeOk bool = false

	for i := 1; i < 5; i++ {
		c, err := net.Dial("tcp", CONNECT)
		if err != nil {
			fmt.Println(err)
			//return
			continue
		}
		fmt.Fprintf(c, message+"\n")
		message, _ := bufio.NewReader(c).ReadString('\n')
		//fmt.Print("->: " + message)
		if strings.TrimSpace(string(message)) == "OK" {
			writeOk = true
			break
		} else {
			fmt.Println("GOT: " + message)
		}
		c.Close()
	}
	if writeOk {
		fmt.Println("TCP socket closing...")
	} else {
		fmt.Println("Could not send message to socket")
	}
}
