package response

import (
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
	writerMode   int
)

type Writer struct {
	mode       writerMode
	headersSet bool
	headers    headers.Headers
	output     io.Writer // in this case write directly to the connection (net.Conn)
	body       []byte
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

const (
	WriterModeBuffered writerMode = iota
	WriterModeChunked
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{headers: headers.NewHeaders(), mode: WriterModeBuffered, output: w}
}

// switches to chunked encoding instead of buffering the response (default)
func (w *Writer) EnableChunkedEncoding() {
	w.mode = WriterModeChunked
	w.headers.Set("Transfer-Encoding", "chunked")
	w.headers.Delete("Content-Length")
}

// SetHeaders merges user-provided headers with the writer's headers.
// User headers take precedence, but protected headers cannot be overridden.
func (w *Writer) SetHeaders(userHeaders headers.Headers) error {
	if w.headersSet {
		return fmt.Errorf("error: headers have already been written")
	}
	// Protected headers that should not be overridden by handlers
	protectedHeaders := map[string]bool{
		"transfer-encoding": w.mode == WriterModeChunked,
		"content-length":    w.mode == WriterModeBuffered,
	}
	// add user headers to the internal headers
	for key, value := range userHeaders {
		formattedKey := strings.ToLower(key)

		if protectedHeaders[formattedKey] {
			log.Printf("Warning: handler attempted to set protected header '%s', ignoring", key)
			continue
		}

		w.headers.Set(key, value)
	}
	return nil
}

func (w *Writer) CheckMode() writerMode {
	return w.mode
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	var line string
	switch statusCode {
	case StatusCodeOK:
		line = "HTTP/1.1 200 OK\r\n"
	case StatusCodeBadRequest:
		line = "HTTP/1.1 400 Bad Request\r\n"
	case StatusCodeInternalServerError:
		line = "HTTP/1.1 500 Internal Server Errror\r\n"
	default:
		log.Println("unsupported status code,leaving the reason phrase blank")
	}
	_, err := w.output.Write([]byte(line))
	return err
}

func (w *Writer) Flush() error {
	if w.mode == WriterModeChunked {
		return fmt.Errorf("error: cannot flush in chunked mode, use WriteChunkedBody() or change the writer mode")
	}
	w.headers.Set("Content-Length", strconv.Itoa(len(w.body)))
	if !w.headersSet {
		if err := w.writeHeaders(); err != nil {
			return err
		}
	}
	_, err := w.output.Write(w.body)
	return err
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.mode != WriterModeChunked {
		return 0, fmt.Errorf("error: the writer must be in chunked mode")
	}
	CRLF := "\r\n"

	// Write headers on first chunk if not already written
	if !w.headersSet {
		if err := w.writeHeaders(); err != nil {
			return 0, err
		}
	}

	// chunked encoding format
	// hex size of data+CRLF
	// chunk data+CRLF
	sizeLine := fmt.Sprintf("%x%s", len(p), CRLF)
	if _, err := w.output.Write([]byte(sizeLine)); err != nil {
		return 0, err
	}
	p = append(p, []byte(CRLF)...)
	if _, err := w.output.Write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *Writer) writeHeaders() error {
	var builder strings.Builder
	for key, value := range w.headers {
		h := fmt.Sprintf("%s: %s\r\n", key, value)
		builder.WriteString(h)
	}
	builder.WriteString("\r\n")
	w.headersSet = true
	_, err := w.output.Write([]byte(builder.String()))

	return err
}

func (w *Writer) WriteChunkedBodyDone() error {
	return w.WriteChunkedBodyDoneWithTrailers(nil)
}

func (w *Writer) WriteChunkedBodyDoneWithTrailers(trailers headers.Headers) error {
	if w.mode != WriterModeChunked {
		return fmt.Errorf("error: the writer must be in chunked mode")
	}

	if _, err := w.output.Write([]byte("0\r\n")); err != nil {
		return err
	}

	return w.writeTrailers(trailers)
}

func (w *Writer) writeTrailers(h headers.Headers) error {
	var builder strings.Builder
	for key, value := range h {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	builder.WriteString("\r\n")
	_, err := w.output.Write([]byte(builder.String()))
	return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.mode == WriterModeChunked {
		return 0, fmt.Errorf("error: cannot WriteBody in chunked mode, use WriteChunkedBody() or switch the writer mode")
	}
	w.body = append(w.body, p...)
	return len(p), nil
}
