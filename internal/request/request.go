package request

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mbeka02/go_http/internal/headers"
)

type Status int

type Request struct {
	RequestLine RequestLine
	Status      Status
	Headers     headers.Headers
	Body        []byte
}

const (
	RequestStateInitialized    Status = iota // 0
	RequestStateDone                         // 1
	RequestStateParsingHeaders               // 2
	RequestStateParsingBody                  // 3
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
var separator = "\r\n"

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
			// log.Println("The buffer is full, allocating more space")
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
	}
	return totalBytesParsed, nil
}

// parseSingle() parses the request line or headers depending on the current state of the parser.
// It returns the number of bytes that are parsed as well as an error
func (r *Request) parseSingle(data []byte) (int, error) {
	var (
		parsedLength int
		err          error
	)
	switch r.Status {
	case RequestStateDone:
		err = fmt.Errorf("error:trying to read data in a done state")
	case RequestStateInitialized:
		requestLine, _, requestLineBytesParsed, parseError := parseRequestLine(string(data))
		// more content is needed before parsing the req line
		if requestLineBytesParsed == 0 {
			break
		}
		// update status and the requestLine
		r.Status = RequestStateParsingHeaders
		r.RequestLine = *requestLine

		parsedLength += requestLineBytesParsed

		err = parseError
	case RequestStateParsingHeaders:
		headersLength, done, parseError := r.Headers.Parse(data)
		parsedLength += headersLength

		if parseError != nil {
			err = parseError
			break
		}
		if done {
			r.Status = RequestStateParsingBody
		}
	case RequestStateParsingBody:
		contentLength := r.Headers.Get("Content-Length")
		// Move to the done state since there's no  request body to parse
		if contentLength == "" {
			r.Status = RequestStateDone
			break
		}
		expectedLength, conversionErr := strconv.Atoi(contentLength)
		if conversionErr != nil {
			err = conversionErr
			break
		}
		if expectedLength < 0 {
			err = fmt.Errorf("invalid Content-Length: cannot be negative")
			break
		}
		currentBodyLength := len(r.Body)
		remainingBodyNeeded := expectedLength - currentBodyLength
		if remainingBodyNeeded <= 0 {
			if remainingBodyNeeded < 0 {
				err = fmt.Errorf("body length (%d) exceeds Content-Length (%d)", currentBodyLength, expectedLength)
				break
			}
			// In this case everything is done
			r.Status = RequestStateDone
			break
		}

		availableData := len(data)
		dataToConsume := remainingBodyNeeded
		// There's not enough data present so just consume what's there
		if availableData < remainingBodyNeeded {
			dataToConsume = availableData
		}
		// Append the data to the body
		r.Body = append(r.Body, data[:dataToConsume]...)
		parsedLength += dataToConsume

		// Terminate if the body is complete
		if len(r.Body) == expectedLength {
			r.Status = RequestStateDone
		} else if len(r.Body) > expectedLength {
			//  safety check
			err = fmt.Errorf("body length (%d) exceeds Content-Length (%d)", len(r.Body), expectedLength)
		}
	default:
		err = fmt.Errorf("invalid state")
	}

	return parsedLength, err
}

func parseRequestLine(s string) (*RequestLine, string, int, error) {
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
	// fmt.Println("HTTP Parts=>", httpParts)
	if len(httpParts) != 2 || httpParts[0] != "HTTP" || httpParts[1] != "1.1" {
		return nil, restOfMessage, lengthParsed, ERROR_MALFORMED_START_LINE
	}

	return &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpParts[1],
	}, restOfMessage, lengthParsed, nil
}
