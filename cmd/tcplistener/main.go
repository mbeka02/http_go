package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	linesChannel := make(chan string)
	go func() {
		// closes the file
		defer f.Close()
		// closes the channel
		defer close(linesChannel)
		fmt.Println("connection has been closed")
		// local variable for the line content
		line := ""
		for {
			b := make([]byte, 8, 8)
			n, err := f.Read(b)
			// handle EOF
			if err != nil {
				if line != "" {
					linesChannel <- line
				}
				if errors.Is(err, io.EOF) {
					break
				}
				log.Printf("read error:%v", err)
				break
			}
			str := string(b[:n])
			parts := strings.Split(str, "\n")
			for i := 0; i < len(parts)-1; i++ {
				linesChannel <- fmt.Sprintf("%s%s", line, parts[i])
				line = ""
			}
			line += parts[len(parts)-1]
		}
	}()
	// return the channel (READ ONLY)
	return linesChannel
}

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("TCP Listen Error:%v", err)
	}
	// Close the listener when exiting
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("TCP Accept Error:%v", err)
		}
		fmt.Println("connection has been accepted")
		linesChannel := getLinesChannel(conn)
		for val := range linesChannel {
			fmt.Println("read:", val)
		}
	}
}
