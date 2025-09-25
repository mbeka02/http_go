package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/mbeka02/go_http/internal/headers"
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

type Handler func(w io.Writer, req *request.Request) *HandlerError

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

// Uses a loop to accept new connections as they come in, and handles each one in a new goroutine. I used an atomic.Bool to track whether the server is closed or not so that I can ignore connection errors after the server is closed.
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
	// parse the request from the connection
	r, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error parsing the request:%v", err)
		respondWithError(conn, "Bad Request", 400)
		return
	}
	buff := new(bytes.Buffer)
	handlerError := s.handler(buff, r)
	if handlerError != nil {
		respondWithError(conn, handlerError.Message, handlerError.StatusCode)
		return
	}
	defaultHeaders := response.GetDefaultHeaders(buff.Len()) // pass the length of the response body
	headers := headers.NewHeaders()
	// write the status line
	err = response.WriteStatusLine(conn, response.StatusCodeOK)
	if err != nil {
		log.Printf("error writing status line:%v", err)
	}
	// add the default headers to the headers map
	for key, value := range defaultHeaders {
		headers[key] = value
	}
	// write the headers
	err = response.WriteHeaders(conn, headers)
	if err != nil {
		log.Printf("error writing headers:%v", err)
	}
	// write the  response body from the handlers buffer
	conn.Write(buff.Bytes())
	defer func() {
		log.Println("...closing the connection")
		conn.Close()
	}()
}
