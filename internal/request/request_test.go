package request

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bufferSize = 8

func RequestFromReader(r io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize, bufferSize)
	var (
		readToIndex int = 0
		bytesParsed int = 0
	)
	request := &Request{
		Status: Initialized,
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
				if request.Status != Done {
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
		fmt.Println("bytes parsed=>", bytesParsed)
		fmt.Println("index =>", readToIndex)
		fmt.Println("request line=>", request.RequestLine)
		copy(buf, buf[bytesParsed:readToIndex])
		readToIndex -= bytesParsed

		// Break when request is complete
		if request.Status == Done {
			break
		}
	}
	fmt.Println("final request line", request.RequestLine)
	return request, nil
}

// parse() accepts the next slice of bytes that needs to be parsed into the Request struct
// It updates the Status(state) of the parser, and the parsed RequestLine field.
// It returns the number of bytes it consumed (meaning successfully parsed) and an error if it encountered one.
func (r *Request) parse(data []byte) (int, error) {
	if r.Status == Done {
		return 0, fmt.Errorf("error:trying to read data in a done state")
	}
	requestLine, _, bytesRead, err := parseRequestLine(string(data))
	if err != nil {
		return bytesRead, err
	}
	// In this scenarion more data is needed before parsing
	if bytesRead == 0 {
		return 0, nil
	}
	// update status and the requestLine
	r.Status = Done
	r.RequestLine = *requestLine
	return bytesRead, nil
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

func TestRequest(t *testing.T) {
	// Test: Good GET Request line
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: Good GET Request line with path
	reader = &chunkReader{
		data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}
