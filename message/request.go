package message

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/mail"
	"time"

	"github.com/tony-montemuro/http/internal/constructs"
)

type AuthorizationCredentials struct {
	Scheme     string
	Parameters map[string]string
}

type ProductVersion struct {
	Product string
	Version string
}

type UserAgent struct {
	Comments []string
	Products []ProductVersion
}

type RequestLine struct {
	Method  Method
	Uri     AbsPathUri
	Version string
}

type RequestHeaders struct {
	Date            MessageTime
	Pragma          PragmaDirectives
	Authorization   AuthorizationCredentials
	From            mail.Address
	IfModifiedSince MessageTime
	Referer         string
	UserAgent       UserAgent
	Allow           []Method
	ContentEncoding ContentEncoding
	ContentLength   ContentLength
	ContentType     ContentType
	Expires         MessageTime
	LastModified    MessageTime
	Unrecognized    map[string]string
	raw             map[string]string
}

type Body []byte

type Request struct {
	Line    RequestLine
	Headers RequestHeaders
	Body    Body
}

type RequestParser struct {
	Connection net.Conn
}

func (p *RequestParser) Parse() (*Request, error) {
	p.Connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer p.Connection.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(p.Connection)

	lineBuf, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if !bytes.HasSuffix(lineBuf, []byte(constructs.Crlf)) {
		return nil, fmt.Errorf("malformed header suffix")
	}

	line, err := requestLineParser(bytes.Trim(lineBuf, constructs.Crlf)).parse()
	if err != nil {
		return nil, err
	}

	var headerBuf bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == "\r\n" {
			break
		}

		headerBuf.WriteString(line)
	}

	headers, err := requestHeadersParser(bytes.Trim(headerBuf.Bytes(), constructs.Crlf)).Parse()
	if err != nil {
		return nil, err
	}

	bodyBytes := make([]byte, headers.ContentLength)
	_, err = io.ReadFull(reader, bodyBytes)
	if err != nil {
		return nil, err
	}

	body, err := requestBodyParser(bodyBytes).parse(headers)
	if err != nil {
		return nil, err
	}

	return &Request{Line: line, Headers: headers, Body: body}, nil
}
