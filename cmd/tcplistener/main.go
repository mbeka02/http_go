package main

import (
	"fmt"
	"log"
	"net"

	"github.com/mbeka02/go_http/internal/request"
)

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
		log.Println("a new connection has been accepted")
		// read and parse data from the connection
		request, err := request.RequestFromReader(conn)
		fmt.Printf("\nRequest line:\n- Method:  %s\n- Target:  %s\n- Version: %s\nHeaders:\n", request.RequestLine.Method, request.RequestLine.RequestTarget, request.RequestLine.HttpVersion)
		for key, value := range request.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}
		fmt.Printf("Body:\n%s", string(request.Body))
	}
}
