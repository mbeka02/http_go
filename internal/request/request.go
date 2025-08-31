package request

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/mbeka02/go_http/internal/headers"
)

type Status int

type Request struct {
	RequestLine RequestLine
	Status      Status
	Headers     headers.Headers
}

const (
	RequestStateInitialized Status = iota // 0
	RequestStateDone                      // 1
	RequestStateParsingHeaders
)
const bufferSize = 8

type RequestLine struct {
	HttpVersion   string
	Method        string
	RequestTarget string
}
type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

var (
	ERROR_MALFORMED_START_LINE  = fmt.Errorf("Malformed Start Line")
	ERROR_INCOMPLETE_START_LINE = fmt.Errorf("The Start Line is incomplete")
)

func (ch *chunkReader) Read(data []byte) (numBytes int, err error) {
	// return if all the data has been read
	if ch.pos >= len(ch.data) {
		return 0, io.EOF
	}
	endIndex := ch.pos + ch.numBytesPerRead
	if endIndex > len(ch.data) {
		endIndex = len(ch.data)
	}

	numBytes = copy(data, ch.data[ch.pos:endIndex])
	ch.pos += numBytes

	return numBytes, nil
}

func RequestFromReader(r io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize, bufferSize)
	var (
		readToIndex int = 0
		bytesParsed int = 0
	)
	request := &Request{
		Status:  RequestStateInitialized,
		Headers: make(map[string]string),
	}
	for {
		// Doubles the buffer size and copies the old content
		if readToIndex == len(buf) {
			log.Println("The buffer is full, allocating more space")
			newSize := len(buf) * 2
			newBuf := make([]byte, newSize)
			copy(newBuf, buf)
			buf = newBuf
		}
		// read into the buffer starting at readToIndex
		bytesRead, err := r.Read(buf[readToIndex:])
		if err != nil {
			// if at the end of the file
			if errors.Is(err, io.EOF) {
				// Parse remaining data before marking as Done
				if readToIndex > 0 {
					bytesParsed, parseErr := request.parse(buf[:readToIndex])
					if parseErr != nil {
						return nil, parseErr
					}
					readToIndex -= bytesParsed
				}
				// Only mark as Done if a full request has been parsed
				if request.Status != RequestStateDone {
					return nil, fmt.Errorf("incomplete request: no complete request line found")
				}
				break
			}
			return nil, fmt.Errorf("unexpected error encountered when reading data: %w", err)
		}
		readToIndex += bytesRead
		// parse the data
		bytesParsed, err = request.parse(buf[:readToIndex])
		if err != nil {
			return nil, err
		}
		// shift the remaining data to the front
		copy(buf, buf[bytesParsed:readToIndex])
		readToIndex -= bytesParsed

		// Break when request is complete
		if request.Status == RequestStateDone {
			break
		}
	}
	return request, nil
}

// parse() accepts the next slice of bytes that needs to be parsed into the Request struct
// It updates the Status(state) of the parser, and the parsed RequestLine field.
// It returns the number of bytes it consumed (meaning successfully parsed) and an error if it encountered one.
func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for r.Status != RequestStateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		// If no progress was made, we need more data - exit the loop
		if n == 0 {
			break
		}
		totalBytesParsed += n
		fmt.Println("Total bytes parsed:", totalBytesParsed)
	}
	// if r.Status == RequestStateDone {
	// 	return 0, fmt.Errorf("error:trying to read data in a done state")
	// }
	// requestLine, _, bytesRead, err := parseRequestLine(string(data))
	// if err != nil {
	// 	return bytesRead, err
	// }
	// // In this scenario more data is needed before parsing
	// if bytesRead == 0 {
	// 	return 0, nil
	// }
	// // update status and the requestLine
	// r.Status = RequestStateDone
	// r.RequestLine = *requestLine
	// return bytesRead, nil
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	var (
		parsedLength int
		err          error
	)
	switch r.Status {
	case RequestStateDone:
		err = fmt.Errorf("error:trying to read data in a done state")
	case RequestStateInitialized:
		log.Printf("...currently parsing the start line , current data: %s", string(data))
		requestLine, _, requestLineLength, parseError := parseRequestLine(string(data))
		if requestLineLength == 0 {
			log.Println("waiting for more data before the request line is parsed")
			break
		}
		// update status and the requestLine
		r.Status = RequestStateParsingHeaders
		r.RequestLine = *requestLine

		parsedLength += requestLineLength

		err = parseError
	case RequestStateParsingHeaders:
		log.Printf("...currently parsing the headers , current data: %s", string(data))
		headersLength, done, parseError := r.Headers.Parse(data)
		parsedLength += headersLength
		err = parseError
		if parseError != nil {
			break // or return error
		}
		if done {
			r.Status = RequestStateDone
		}
	default:
		err = fmt.Errorf("invalid parser state")
	}

	return parsedLength, err
}

func parseRequestLine(s string) (*RequestLine, string, int, error) {
	separator := "\r\n"
	idx := strings.Index(s, separator)
	// If there are no occurences of the separator in s do an early return
	// This just means that more data is needed before parsing the request line.
	if idx == -1 {
		return nil, s, 0, nil
	}
	// get the start line
	startLine := s[:idx]
	// include CRLF in bytesRead
	lengthParsed := len([]byte(startLine)) + len(separator)
	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, s, lengthParsed, ERROR_MALFORMED_START_LINE
	}
	restOfMessage := s[idx+len(separator):]
	httpParts := strings.Split(parts[2], "/")
	if len(httpParts) != 2 || httpParts[0] != "HTTP" || httpParts[1] != "1.1" {
		return nil, restOfMessage, lengthParsed, ERROR_MALFORMED_START_LINE
	}

	return &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpParts[1],
	}, restOfMessage, lengthParsed, nil
}
