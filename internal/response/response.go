package response

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/mbeka02/go_http/internal/headers"
)

type StatusCode int

const (
	StatusCodeOK StatusCode = iota
	StatusCodeBadRequest
	StatusCodeInternalServerError
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	var err error
	switch statusCode {
	case StatusCodeOK:
		n, writeErr := w.Write([]byte("HTTP/1.1 200 OK\r\n"))
		err = writeErr
		log.Println("Written:", n, "bytes to the connection")
	case StatusCodeBadRequest:
		_, err = w.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
	case StatusCodeInternalServerError:
		_, err = w.Write([]byte("HTTP/1.1 500 Internal Server Errror\r\n"))
	default:
		log.Println("unsupported status code , leaving the reason phrase blank")
	}
	return err
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	var (
		builder strings.Builder
		result  string
	)
	for key, value := range headers {
		headerText := fmt.Sprintf("%s:%s\r\n", key, value)
		builder.WriteString(headerText)

	}
	builder.WriteString("\r\n")
	result = builder.String()
	n, err := w.Write([]byte(result))
	log.Println("Written:", n, "bytes to the connection")

	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	contentLenStr := strconv.Itoa(contentLen)
	headers := headers.NewHeaders()
	headers.Set("Content-Length", contentLenStr)
	headers.Set("Connection", "close")
	headers.Set("Content-Type", "text/plain")
	return headers
}
