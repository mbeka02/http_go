package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/mbeka02/go_http/internal/request"
	"github.com/mbeka02/go_http/internal/response"
)

type Server struct {
	listener net.Listener
	handler  Handler
	closed   atomic.Bool
}
type HandlerError struct {
	Message    string
	StatusCode int
}

type Handler func(w *response.Writer, req *request.Request)

func respondWithError(w io.Writer, message string, statusCode int) error {
	var statusLine string
	switch statusCode {
	case 400:
		statusLine = "HTTP/1.1 400 Bad Request\r\n"
	case 500:
		statusLine = "HTTP/1.1 500 Internal Server Error\r\n"
	default:
		statusLine = fmt.Sprintf("HTTP/1.1 %d Unknown Error\r\n", statusCode)
	}

	body := []byte(message)
	contentLength := len(body)
	headers := fmt.Sprintf(
		"Content-Length: %d\r\nConnection: close\r\nContent-Type: text/plain\r\n\r\n",
		contentLength,
	)

	if _, err := w.Write([]byte(statusLine)); err != nil {
		return err
	}
	if _, err := w.Write([]byte(headers)); err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}

	return nil
}

// Creates a net.Listener and returns a new Server instance. Starts listening for requests inside a goroutine.
func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, fmt.Errorf("TCP Listen Error:%v", err)
	}

	server := &Server{listener: listener, handler: handler}
	go server.listen()
	return server, nil
}

// Closes the listener and the server
func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

// Uses a loop to accept new connections as they come in, and handles each one in a new goroutine.
func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Go back to the top of the for loop and try Accept() again
			continue
		}
		go s.handle(conn)
	}
}

// Handles a single connection by writing the  response and then closing the connection
func (s *Server) handle(conn net.Conn) {
	log.Printf("Handling connection from %s", conn.RemoteAddr())
	defer func() {
		log.Println("...closing the connection")
		conn.Close()
	}()
	// parse the request from the connection
	r, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error parsing the request:%v", err)
		respondWithError(conn, "Bad Request", 400)
		return
	}
	writer := response.NewWriter(conn)

	s.handler(writer, r)
	if writer.CheckMode() == response.WriterModeChunked {
		log.Println("... writer is in chunked mode")
	} else {
		// flush  writes the headers and body to the buffer
		log.Println("... writer is in buffered mode , flushing the headers and response body")
		if err := writer.Flush(); err != nil {
			log.Printf("error flushing writer: %v", err)
			respondWithError(conn, "Internal Server Error", 500)
			return
		}
	}
}
