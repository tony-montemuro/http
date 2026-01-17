package message

import (
	"fmt"
	"time"
)

type Method string

const (
	MethodGet  Method = "GET"
	MethodHead Method = "HEAD"
	MethodPost Method = "POST"
)

func (m Method) Validate() error {
	switch m {
	case MethodGet, MethodHead, MethodPost:
		return nil
	}
	return fmt.Errorf("invalid method")
}

type ContentEncoding string

const (
	ContentEncodingXGzip     = "x-gzip"
	ContentEncodingXCompress = "x-compress"
)

func (e ContentEncoding) Validate() error {
	switch e {
	case ContentEncodingXGzip, ContentEncodingXCompress:
		return nil
	}
	return fmt.Errorf("unknown encoding")
}

type ContentLength uint64

type MessageTime struct {
	date time.Time
}

type PragmaDirectives struct {
	Flags   []string
	Options map[string]string
}

type ContentType struct {
	Type       string
	Subtype    string
	Parameters map[string]string
}
