package http

import (
	"net/mail"
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
	Uri     RelativeUri
	Version string
}

type RequestHeaders struct {
	Date            MessageTime
	Pragma          PragmaDirectives
	Authorization   AuthorizationCredentials
	From            mail.Address
	IfModifiedSince MessageTime
	Referer         Uri
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

func (r Request) GetRawHeader(name string) (string, bool) {
	value, ok := r.Headers.raw[name]
	return value, ok
}
