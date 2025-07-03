package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	addr := "localhost:42069"
	UDPAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("unable to resolve addr:%v", err)
	}
	conn, err := net.DialUDP("udp", nil, UDPAddr)
	if err != nil {
		log.Fatalf("UDP Connection Error:%v", err)
	}

	defer conn.Close()
	fmt.Printf("Type in your message and press enter to send to:%s", addr)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("IO Error , unable to read input : %v", err)
		}
		// send the mesage as a stream of bytes
		_, err = conn.Write([]byte(message))
		if err != nil {
			log.Fatalf("UDP Write Error:%v", err)
		}
		fmt.Printf("Message sent:%v", message)
	}
}
