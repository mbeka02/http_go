package headers

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
)

type Headers map[string]string

var (
	ERROR_INVALID_FIELD_LINE = fmt.Errorf("the field line is invalid , unable to add it to the parsed headers")
	ERROR_INVALID_FIELD_NAME = fmt.Errorf("the field name is invalid , it likely contains whitespace")
	ERROR_MISSING_HEADER_KEY = fmt.Errorf("header does not exist")
	ERROR_INVALID_HEADER_KEY = fmt.Errorf("The header key contains an invalid character")
)
var fieldNameRegex = regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*+\-.^_` + "`" + `|~]+$`)

func NewHeaders() Headers {
	h := make(map[string]string)
	return h
}

func (h Headers) Get(key string) string {
	formattedKey := strings.ToLower(key)
	val, ok := h[formattedKey]
	if !ok {
		log.Println(ERROR_MISSING_HEADER_KEY)

		return ""
	}
	return val
}

func (h Headers) Set(key, value string) {
	formattedKey := strings.ToLower(key)
	h[formattedKey] = value
}

func (h Headers) List() {
	for key, val := range h {
		fmt.Println("Key:", key, "Value:", val)
	}
}

// Parse() reads data and calls parseHeader() that adds individual headers to the map// it returns the amount of data read , the parser state and an error
func (h Headers) Parse(data []byte) (int, bool, error) {
	crlf := []byte("\r\n")
	dataRead := 0
	done := false
	for {
		idx := bytes.Index(data[dataRead:], crlf)
		// assume you haven't been given enough data yet.
		if idx == -1 {
			break
		}

		// if the CRLF is at the beginning then it's an empty header ,
		if idx == 0 {
			done = true
			dataRead += len(crlf)
			break
		}

		fieldName, fieldValue, err := parseHeader(data[dataRead : dataRead+idx])
		if err != nil {
			log.Println(err)
			return dataRead, done, err
		}
		dataRead += idx + len(crlf)
		// fmt.Println("read:", dataRead, "bytes")
		// add the pair to the headers map
		val, ok := h[fieldName]
		if ok {
			// concat the new value if the key already exists
			h[fieldName] = val + "," + fieldValue
		} else {
			h[fieldName] = fieldValue
		}
	}
	return dataRead, done, nil
}

// A helper function that parses individual field lines
func parseHeader(fieldLine []byte) (string, string, error) {
	pair := bytes.SplitN(fieldLine, []byte(":"), 2)
	// for _, val := range pair {
	// 	fmt.Println("pair=>", string(val))
	// }
	if len(pair) != 2 {
		return "", "", ERROR_INVALID_FIELD_LINE
	}
	fieldName := pair[0]
	fieldValue := bytes.TrimSpace(pair[1])
	// Ensure there's no whitespace betwixt the field name and colon
	if bytes.HasSuffix(fieldName, []byte(" ")) {
		return "", "", ERROR_INVALID_FIELD_NAME
	}
	formattedFieldName := strings.TrimSpace(strings.ToLower(string(fieldName)))
	// ensure the field name uses valid characters
	match := fieldNameRegex.MatchString(formattedFieldName)
	if !match {
		return "", "", ERROR_INVALID_HEADER_KEY
	}
	formattedFieldValue := string(fieldValue)
	return formattedFieldName, formattedFieldValue, nil
}
