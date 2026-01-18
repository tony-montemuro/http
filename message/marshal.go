package message

import (
	"bytes"
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

func (c code) marshal() []byte {
	return fmt.Appendf([]byte{}, "HTTP/1.0 %d %s%s", c, StatusText(int(c)), constructs.Crlf)
}

func (h responseHeaders) marshal(hasBody bool) []byte {
	var headers []byte

	headers = append(headers, marshalHeader("Date", h.Date)...)
	headers = append(headers, marshalHeader("Pragma", h.Pragma)...)
	if h.Location != nil {
		headers = append(headers, marshalHeader("Location", h.Location)...)
	}

	headers = append(headers, marshalHeader("Server", h.Server)...)
	headers = append(headers, marshalHeader("WWW-Authenticate", h.WwwAuthenticate)...)
	headers = append(headers, marshalHeader("Allow", h.Allow)...)
	headers = append(headers, marshalHeader("Content-Encoding", h.ContentEncoding)...)

	if hasBody {
		headers = append(headers, marshalHeader("Content-Length", h.ContentLength)...)
	}

	headers = append(headers, marshalHeader("Content-Type", h.ContentType)...)
	headers = append(headers, marshalHeader("Expires", h.Expires)...)
	headers = append(headers, marshalHeader("Last-Modified", h.LastModified)...)

	for _, name := range getSortedKeys(h.Unrecognized) {
		headers = fmt.Appendf(headers, "%s: %s%s", name, h.Unrecognized[name], constructs.Crlf)
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
		for _, flag := range p.Flags {
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

func getSortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
