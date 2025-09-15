package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	closed   atomic.Bool
}

// Creates a net.Listener and returns a new Server instance. Starts listening for requests inside a goroutine.
func Serve(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, fmt.Errorf("TCP Listen Error:%v", err)
	}

	server := &Server{listener: listener}
	go server.listen()
	return server, nil
}

// Closes the listener and the server
func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

// Uses a loop to .Accept new connections as they come in, and handles each one in a new goroutine. I used an atomic.Bool to track whether the server is closed or not so that I can ignore connection errors after the server is closed.
func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			// Only log error if server hasn't been closed
			if !s.closed.Load() {
				log.Printf("TCP accept() error: %s", err)
			}
			// continue - go back to the top of the for loop and try Accept() again
			continue
		}
		go s.handle(conn)
	}
}

// Handles a single connection by writing the following response and then closing the connection
func (s *Server) handle(conn net.Conn) {
	log.Printf("Handling connection from %s", conn.RemoteAddr())
	_, err := conn.Write([]byte(`
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 13
Hello World!`))
	if err != nil {
		log.Printf("write err:%v", err)
	}

	defer func() {
		log.Println("...closing the connection")
		conn.Close()
	}()
}
