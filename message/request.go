package message

import (
	"net"
	"net/mail"
	"time"

	"github.com/tony-montemuro/http/internal/parser"
)

type Method string

const (
	MethodGet  Method = "GET"
	MethodHead Method = "HEAD"
	MethodPost Method = "POST"
)

func (m Method) IsValid() bool {
	switch m {
	case MethodGet, MethodHead, MethodPost:
		return true
	}
	return false
}

type ContentEncoding string

const (
	ContentEncodingXGzip     ContentEncoding = "x-gzip"
	ContentEncodingXCompress ContentEncoding = "x-compress"
)

type RequestLine struct {
	Method  Method
	Uri     AbsPathUri
	Version string
}

type PragmaDirectives struct {
	Flags   []string
	Options map[string]string
}

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

type ContentType struct {
	Type       string
	Subtype    string
	Parameters map[string]string
}

type RequestHeaders struct {
	Date            time.Time
	Pragma          PragmaDirectives
	Authorization   AuthorizationCredentials
	From            mail.Address
	IfModifiedSince time.Time
	Referer         string
	UserAgent       UserAgent
	Allow           []Method
	ContentEncoding ContentEncoding
	ContentLength   uint64
	ContentType     ContentType
	Expires         time.Time
	LastModified    time.Time
	Unrecognized    map[string]string
	raw             map[string]string
}

type Body []byte

type Request struct {
	Line    RequestLine
	Headers RequestHeaders
	Body    Body
}

type AbsPathUri struct {
	Path   [][]byte
	Params [][]byte
	Query  []byte
}

type RequestParser struct {
	Connection net.Conn
}

func (p *RequestParser) Parse() (*Request, error) {
	data := make([]byte, 1024)
	_, err := p.Connection.Read(data)

	if err != nil {
		return nil, err
	}

	parsedRequest, err := parser.RequestParser(data).Parse()
	if err != nil {
		return nil, err
	}

	return &Request{
		Line:    convertToRequestLine(parsedRequest.Line),
		Headers: convertToRequestHeader(parsedRequest.Headers),
		Body:    convertToRequestBody(parsedRequest.Body),
	}, nil
}

func convertToRequestLine(parsed parser.ParsedRequestLine) RequestLine {
	return RequestLine{
		Method: Method(parsed.Method),
		Uri: AbsPathUri{
			Path:   parsed.Uri.Path,
			Params: parsed.Uri.Params,
			Query:  parsed.Uri.Query,
		},
		Version: string(parsed.Version),
	}
}

func convertToRequestHeader(parsed parser.ParsedRequestHeaders) RequestHeaders {
	headers := RequestHeaders{
		Date: parsed.Date,
		Pragma: PragmaDirectives{
			Flags:   parsed.Pragma.Flags,
			Options: parsed.Pragma.Options,
		},
		Authorization: AuthorizationCredentials{
			Scheme:     parsed.Authorization.Scheme,
			Parameters: parsed.Authorization.Parameters,
		},
		From:    parsed.From,
		Referer: parsed.Referer,
		UserAgent: UserAgent{
			Comments: parsed.UserAgent.Comments,
			Products: []ProductVersion{},
		},
		IfModifiedSince: parsed.IfModifiedSince,
		ContentEncoding: ContentEncoding(parsed.ContentEncoding),
		ContentLength:   parsed.ContentLength,
		ContentType:     ContentType(parsed.ContentType),
		Expires:         parsed.Expires,
		LastModified:    parsed.LastModified,
		Unrecognized:    parsed.Unrecognized,
		raw:             parsed.Raw,
	}

	for _, product := range parsed.UserAgent.Products {
		headers.UserAgent.Products = append(headers.UserAgent.Products, ProductVersion{
			Product: product.Product,
			Version: product.Version,
		})
	}

	for _, method := range parsed.Allow {
		headers.Allow = append(headers.Allow, Method(method))
	}

	return headers
}

func convertToRequestBody(body []byte) Body {
	return Body(body)
}
