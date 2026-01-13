package parser

import (
	"bytes"
	"fmt"
)

type ParsedRequest struct {
	Line    ParsedRequestLine
	Headers ParsedRequestHeaders
	Body    []byte
}

type RequestParser []byte

func (r RequestParser) Parse() (ParsedRequest, error) {
	crlfIndex := bytes.Index(r, []byte(crlf))
	if crlfIndex == 1 {
		return ParsedRequest{}, ClientError{message: fmt.Sprintf("Incomplete request (%s)", string(r))}
	}

	line, err := requestLineParser(r[:crlfIndex]).Parse()
	if err != nil {
		return ParsedRequest{}, err
	}

	headerDelim := append([]byte(crlf), []byte(crlf)...)
	headerEndIndex := bytes.Index(r, headerDelim)
	if headerEndIndex == -1 {
		return ParsedRequest{}, ClientError{message: fmt.Sprintf("Incomplete request (%s)", string(r))}
	}

	headers, err := requestHeadersParser(r[min(crlfIndex+len(crlf), headerEndIndex):headerEndIndex]).Parse()
	if err != nil {
		return ParsedRequest{}, err
	}
	body, err := requestBodyParser(r[headerEndIndex+len(headerDelim):]).Parse(headers)
	if err != nil {
		return ParsedRequest{}, err
	}

	return ParsedRequest{Line: line, Headers: headers, Body: body}, nil
}
