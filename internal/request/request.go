package request

import (
	"fmt"
	"io"
)

type Status int

type Request struct {
	RequestLine RequestLine
	Status      Status
}

const (
	Initialized Status = iota // 0
	Done                      // 1
)

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

// func (ch *chunkReader) Read(data []byte) (numBytes int, err error) {
// 	// return if all the data has been read
// 	if ch.pos >= len(ch.data) {
// 		return 0, io.EOF
// 	}
//
// 	endIndex := ch.pos + ch.numBytesPerRead
// 	if endIndex > len(ch.data) {
// 		endIndex = len(ch.data)
// 	}
// 	numBytes = copy(data, ch.data[ch.pos:endIndex])
// 	ch.pos += numBytes
// 	// FIXME
// 	if numBytes > ch.numBytesPerRead {
// 		numBytes = ch.numBytesPerRead
// 		// ISN'T THIS KIND OF REDUNDANT?
// 		ch.pos -= numBytes - ch.numBytesPerRead
// 	}
// 	return numBytes, nil
// }

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
