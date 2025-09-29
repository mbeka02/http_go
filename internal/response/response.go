package response

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/mbeka02/go_http/internal/headers"
)

type (
	StatusCode   int
	writerStatus int
)

type Writer struct {
	buffer  *bytes.Buffer
	headers headers.Headers
	body    []byte
}

const (
	StatusCodeOK StatusCode = iota
	StatusCodeBadRequest
	StatusCodeInternalServerError
)

const (
	WriterStatusInitialized writerStatus = iota
	WriterStatusStatusLine
	WriterStatusHeaders
	WriterStatusBody
	WriterStatusDone
)

func NewWriter(buffer *bytes.Buffer) *Writer {
	return &Writer{buffer: buffer, headers: headers.NewHeaders()}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	var err error
	switch statusCode {
	case StatusCodeOK:
		n, writeErr := w.buffer.Write([]byte("HTTP/1.1 200 OK\r\n"))
		err = writeErr
		log.Println("Written:", n, "bytes to the buffer")
	case StatusCodeBadRequest:
		_, err = w.buffer.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
	case StatusCodeInternalServerError:
		_, err = w.buffer.Write([]byte("HTTP/1.1 500 Internal Server Errror\r\n"))
	default:
		log.Println("unsupported status code,leaving the reason phrase blank")
	}
	return err
}

func (w *Writer) Flush() error {
	w.headers.Set("Content-Length", strconv.Itoa(len(w.body)))

	// write headers
	var builder strings.Builder
	for key, value := range w.headers {
		h := fmt.Sprintf("%s: %s\r\n", key, value)
		builder.WriteString(h)
	}
	builder.WriteString("\r\n")
	w.buffer.Write([]byte(builder.String()))

	// write body
	_, err := w.buffer.Write(w.body)
	return err
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	w.headers = headers
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
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
