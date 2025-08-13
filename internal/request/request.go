package request

import "fmt"

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HttpVersion   string
	Method        string
	RequestTarget string
}

var (
	ERROR_INVALID_START_LINE    = fmt.Errorf("Invalid Start Line")
	ERROR_INCOMPLETE_START_LINE = fmt.Errorf("The Start Line is incomplete")
)
