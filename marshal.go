package http

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tony-montemuro/http/internal/constructs"
)

type marshaler interface {
	marshal() []byte
}

func (r response) marshal() []byte {
	var marshaled []byte

	line := r.code.marshal()
	marshaled = append(marshaled, line...)

	headers := r.headers.marshal(len(r.body) > 0)
	marshaled = append(marshaled, headers...)

	marshaled = append(marshaled, r.body...)
	return marshaled
}

func (c code) marshal() []byte {
	return fmt.Appendf([]byte{}, "HTTP/1.0 %d %s%s", c, StatusText(int(c)), constructs.Crlf)
}

func (h responseHeaders) marshal(hasBody bool) []byte {
	var headers []byte

	headers = append(headers, marshalHeader("Date", h.date)...)
	headers = append(headers, marshalHeader("Pragma", h.pragma)...)

	if h.location != nil {
		headers = append(headers, marshalHeader("Location", h.location)...)
	}

	headers = append(headers, marshalHeader("Server", h.server)...)
	headers = append(headers, marshalHeader("WWW-Authenticate", h.wwwAuthenticate)...)
	headers = append(headers, marshalHeader("Allow", h.allow)...)
	headers = append(headers, marshalHeader("Content-Encoding", h.contentEncoding)...)

	if hasBody {
		headers = append(headers, marshalHeader("Content-Length", h.contentLength)...)
	}

	headers = append(headers, marshalHeader("Content-Type", h.contentType)...)
	headers = append(headers, marshalHeader("Expires", h.expires)...)
	headers = append(headers, marshalHeader("Last-Modified", h.lastModified)...)

	for _, name := range getSortedKeys(h.unrecognized) {
		headers = fmt.Appendf(headers, "%s: %s%s", name, h.unrecognized[name], constructs.Crlf)
	}

	return append(headers, constructs.Crlf...)
}

func marshalHeader(n string, m marshaler) []byte {
	s := m.marshal()

	if len(s) == 0 {
		return s
	}

	return fmt.Appendf([]byte{}, "%s: %s%s", n, s, constructs.Crlf)

}

func (t MessageTime) marshal() []byte {
	var res []byte

	if !t.date.IsZero() {
		res = []byte(t.date.Format(time.RFC1123))
	}

	return res
}

func (p PragmaDirectives) marshal() []byte {
	var parts []string

	if len(p.Flags) > 0 || len(p.Options) > 0 {
		for _, flag := range getSortedKeys(p.Flags) {
			parts = append(parts, flag)
		}

		for _, name := range getSortedKeys(p.Options) {
			parts = append(parts, fmt.Sprintf("%s=%s", name, p.Options[name]))
		}
	}

	return []byte(strings.Join(parts, " "))
}

func (pv ProductVersion) marshal() []byte {
	res := []byte(pv.Product)

	if len(pv.Version) > 0 {
		res = append(res, fmt.Sprintf("/%s", pv.Version)...)
	}

	return res
}

func (u AbsoluteUri) marshal() []byte {
	return fmt.Appendf([]byte{}, "%s:%s", u.Scheme, u.Path)
}

func (u RelativeUri) marshal() []byte {
	var res []byte

	if len(u.NetLoc) > 0 {
		res = fmt.Appendf(res, "//%s", u.NetLoc)
	}

	res = append(res, u.Path...)
	if len(u.Params) > 0 {
		joined := bytes.Join(u.Params, []byte{';'})
		res = fmt.Appendf(res, ";%s", joined)
	}

	if len(u.Query) > 0 {
		res = fmt.Appendf(res, "?%s", u.Query)
	}

	return res
}

func (s server) marshal() []byte {
	var parts []string

	if len(s.products) > 0 || len(s.comments) > 0 {
		for _, product := range s.products {
			parts = append(parts, string(product.marshal()))
		}

		for _, comment := range s.comments {
			parts = append(parts, comment)
		}
	}

	return []byte(strings.Join(parts, " "))
}

func (c challenge) marshal() []byte {
	var res []byte

	if len(c.scheme) > 0 && len(c.realm) > 0 {
		res = fmt.Appendf([]byte{}, "%s realm=%s", c.scheme, c.realm)
	}

	for _, name := range getSortedKeys(c.params) {
		res = fmt.Appendf(res, ",%s=%s", name, c.params[name])
	}

	return res
}

func (m Methods) marshal() []byte {
	methods := make([]string, len(m.methods))

	for i, method := range m.methods {
		methods[i] = string(method)
	}

	return []byte(strings.Join(methods, ", "))
}

func (ce ContentEncoding) marshal() []byte {
	var res []byte

	if len(ce) > 0 {
		res = append(res, []byte(ce)...)
	}

	return res
}

func (cl ContentLength) marshal() []byte {
	return []byte(strconv.FormatUint(uint64(cl), 10))
}

func (ct ContentType) marshal() []byte {
	var res []byte

	if len(ct.Type) > 0 && len(ct.Subtype) > 0 {
		res = fmt.Appendf(res, "%s/%s", ct.Type, ct.Subtype)
	}

	for _, name := range getSortedKeys(ct.Parameters) {
		res = fmt.Appendf(res, ";%s=%s", name, ct.Parameters[name])
	}

	return res
}

func getSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func encodeRequestBody(body []byte, encoding ContentEncoding) ([]byte, error) {
	var res []byte
	var err error

	switch encoding {
	case ContentEncodingXGzip, ContentEncodingGZip:
		res, err = gzipEncode(body)
	case ContentEncodingXCompress, ContentEncodingCompress:
		res, err = compressEncode(body)
	default:
		res, err = body, nil
	}

	if err != nil {
		err = ServerError{message: fmt.Sprintf("unexpected issue decoding body: %s", err.Error())}
	}

	return res, err
}

func gzipEncode(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(data)
	if err != nil {
		return b.Bytes(), err
	}

	err = w.Close()
	return b.Bytes(), err
}

func compressEncode(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := lzw.NewWriter(&b, lzw.LSB, 8)

	_, err := w.Write(data)
	if err != nil {
		return b.Bytes(), err
	}

	err = w.Close()
	return b.Bytes(), err
}
