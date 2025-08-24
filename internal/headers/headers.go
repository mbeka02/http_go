package headers

import (
	"fmt"
	"strings"
	"unicode"
)

type Headers map[string]string

var (
	ERROR_INVALID_FIELD_LINE = fmt.Errorf("the field line is invalid , unable to add it to the parsed headers")
	ERROR_INVALID_FIELD_NAME = fmt.Errorf("the field name is invalid , it likely contains whitespace")
)

func NewHeaders() Headers {
	h := make(map[string]string)
	return h
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	crlf := "\r\n"
	dataStr := string(data)
	//???
	n = len(dataStr) - len(crlf)
	idx := strings.Index(dataStr, crlf)
	// assume you haven't been given enough data yet.
	if idx == -1 {
		return 0, false, nil
	}

	// if the CRLF is at the beginning then we're at the end of the headers ,
	// return all the data immediately
	if strings.HasPrefix(dataStr, crlf) {
		return 0, true, nil
	}
	// Remove whitespace on either side of the string
	dataStr = strings.TrimSpace(dataStr)
	// Extract the key value pair
	colonIndex := strings.Index(dataStr, ":")
	// Invalid key value pair
	if colonIndex == -1 {
		return 0, false, ERROR_INVALID_FIELD_LINE
	}
	fieldName := dataStr[:colonIndex]
	fieldValue := dataStr[colonIndex+1:]
	// Ensure there's no whitespace in the field name
	if containsWhitespace(fieldName) {
		return 0, false, ERROR_INVALID_FIELD_LINE
	}
	// add the pair to the headers map
	h[fieldName] = strings.TrimSpace(fieldValue)
	return n, false, nil
}

// containsWhitespace() is a helper function that checks for any whitespace
func containsWhitespace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
