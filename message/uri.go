package message

import (
	"bytes"
	"fmt"

	"github.com/tony-montemuro/http/internal/constructs"
)

type Uri interface {
	GetPath() []byte
	marshal() []byte
}

type escapeSequence []byte

func (s escapeSequence) unescape(i int) (byte, error) {
	var b byte

	for j := 1; j <= 2; j++ {
		if i+j == len(s) {
			return b, ClientError{message: fmt.Sprintf("truncated escape sequence: (char pos: %d, \"%s\")", i+j-1, s)}
		}

		val, err := constructs.Hex(s[i+j]).Value()
		if err != nil {
			return b, ClientError{message: fmt.Sprintf("malformed escape sequence: (char pos: %d, \"%s\")", i+j, s[:i+j])}
		}

		b += (val) << (4 * (2 - j))
	}

	return b, nil
}

type AbsoluteUri struct {
	Scheme []byte
	Path   []byte
}

func (u AbsoluteUri) GetPath() []byte {
	return u.marshal()
}

type absoluteUriParser []byte

func (a absoluteUriParser) Parse() (AbsoluteUri, error) {
	var uri AbsoluteUri
	scheme, remaining, found := bytes.Cut(a, []byte{':'})
	if !found {
		return uri, fmt.Errorf("could not determine scheme")
	}

	err := constructs.Scheme(scheme).Validate()
	if err != nil {
		return uri, err
	}
	uri.Scheme = scheme

	var path []byte
	i := 0

	for i < len(remaining) {
		b := constructs.HttpByte(remaining[i])

		if b.IsEscape() {
			c, err := escapeSequence(remaining).unescape(i)
			if err != nil {
				return uri, err
			}
			i += 3
			b = constructs.HttpByte(c)
		} else {
			i++
		}

		if !b.IsReserved() && !b.IsUnreserved() {
			return uri, fmt.Errorf("queries contain invalid byte (%s)", remaining)
		}

		path = append(path, byte(b))
	}

	uri.Path = path
	return uri, nil
}

type RelativeUri struct {
	NetLoc []byte
	Path   []byte
	Params [][]byte
	Query  []byte
}

func (u RelativeUri) GetPath() []byte {
	return u.marshal()
}

const (
	NetPath = "net_path"
	AbsPath = "abs_path"
	RelPath = "rel_path"
)

func (u RelativeUri) getPathForm() string {
	if len(u.NetLoc) > 0 {
		return NetPath
	}
	if len(u.Path) == 0 || u.Path[0] != constructs.ByteSeparator {
		return RelPath
	}
	return AbsPath
}

type relativeUriParser []byte

func (r relativeUriParser) parse() (RelativeUri, error) {
	uri := RelativeUri{}
	start := 0

	if len(r) >= 2 && r[0] == constructs.ByteSeparator && r[1] == constructs.ByteSeparator {
		i := 2

		for i < len(r) && (constructs.HttpByte(r[i]).IsPChar() || r[i] == ';' || r[i] == '?') {
			i++
		}

		uri.NetLoc = r[2:i]
		start = i
	}

	if start == len(r) {
		return uri, nil
	}

	var err error
	var path, query []byte
	var params [][]byte

	if start > 0 || r[start] == constructs.ByteSeparator {
		path, params, query, err = absPathUriParser(r[start:]).parse()
	} else {
		path, params, query, err = relPathUriParser(r[start:]).parse()
	}

	if err != nil {
		return uri, err
	}

	uri.Path = path
	uri.Params = params
	uri.Query = query

	return uri, nil
}

type absPathUriParser []byte

func (rp absPathUriParser) parse() ([]byte, [][]byte, []byte, error) {
	var path, query []byte
	var params [][]byte
	var err error = fmt.Errorf("abs_path must begin with /")

	if len(rp) == 0 || rp[0] != constructs.ByteSeparator {
		return path, params, query, err
	}

	path, params, query, err = relPathUriParser(rp[1:]).parse()
	return append([]byte("/"), path...), params, query, err
}

type relPathUriParser []byte

func (ap relPathUriParser) parse() ([]byte, [][]byte, []byte, error) {
	var path, query []byte
	var params [][]byte

	paramsIndex := bytes.IndexByte(ap, constructs.ByteParam)
	queryIndex := bytes.IndexByte(ap, constructs.ByteQuery)

	var paramsSlice []byte
	var querySlice []byte

	if queryIndex != -1 {
		querySlice = ap[queryIndex+1:]
	} else {
		queryIndex = len(ap)
	}

	if paramsIndex != -1 && paramsIndex < queryIndex {
		paramsSlice = ap[paramsIndex+1 : queryIndex]
	} else {
		paramsIndex = queryIndex
	}

	path, err := uriPathParser(ap[:paramsIndex]).parse()
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri path: %s", err)}
	}

	params, err = uriParamsParser(paramsSlice).parse()
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri param(s): %s", err)}
	}

	query, err = uriQueryParser(querySlice).parse()
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri querie(s): %s", err)}
	}

	return path, params, query, nil
}

type uriPathParser []byte

func (p uriPathParser) parse() ([]byte, error) {
	var path [][]byte
	var res []byte
	unescaped := bytes.Split(p, []byte{byte(constructs.ByteSeparator)})

	// special case: if we have at least 1 segment, the first segment cannot be empty according to RFC 1945 (see: https://datatracker.ietf.org/doc/html/rfc1945#section-3.2.1)
	if len(unescaped) > 1 && len(unescaped[0]) == 0 {
		return res, fmt.Errorf("first segment cannot be empty")
	}

	for _, p := range unescaped {
		j := 0
		var part []byte

		for j < len(p) {
			b := constructs.HttpByte(p[j])

			if b.IsEscape() {
				c, err := escapeSequence(p).unescape(j)
				if err != nil {
					return res, err
				}
				j += 3
				b = constructs.HttpByte(c)
			} else {
				j++
			}

			if !b.IsPChar() {
				return res, fmt.Errorf("path contains invalid byte (%s)", p)
			}

			part = append(part, byte(b))
		}

		path = append(path, part)
	}

	res = bytes.Join(path, []byte{'/'})
	return res, nil
}

type uriParamsParser []byte

func (p uriParamsParser) parse() ([][]byte, error) {
	var params [][]byte

	for p := range bytes.SplitSeq(p, []byte{byte(constructs.ByteParam)}) {
		j := 0
		var param []byte

		for j < len(p) {
			b := constructs.HttpByte(p[j])

			if b.IsEscape() {
				c, err := escapeSequence(p).unescape(j)
				if err != nil {
					return params, err
				}
				j += 3
				b = constructs.HttpByte(c)
			} else {
				j++
			}

			if !b.IsPChar() && b != constructs.ByteSeparator {
				return params, fmt.Errorf("params contains invalid byte (%s)", p)
			}

			param = append(param, byte(b))
		}

		params = append(params, param)
	}

	return params, nil
}

type uriQueryParser []byte

func (q uriQueryParser) parse() ([]byte, error) {
	var query []byte
	i := 0

	for i < len(q) {
		b := constructs.HttpByte(q[i])

		if b.IsEscape() {
			c, err := escapeSequence(q).unescape(i)
			if err != nil {
				return query, err
			}
			i += 3
			b = constructs.HttpByte(c)
		} else {
			i++
		}

		if !b.IsReserved() && !b.IsUnreserved() {
			return query, fmt.Errorf("queries contain invalid byte (%s)", q)
		}

		query = append(query, byte(b))
	}

	return query, nil
}

type safeUriParser string

func (u safeUriParser) parse() (string, error) {
	var uri []byte
	i := 0

	for i < len(u) {
		b := constructs.HttpByte(u[i])

		if b.IsEscape() {
			c, err := escapeSequence(u).unescape(i)
			if err != nil {
				return string(u), err
			}
			i += 3
			b = constructs.HttpByte(c)
		} else {
			i++
		}

		if b.IsUnsafe() && b != '#' {
			return string(u), fmt.Errorf("uri contains at least 1 unsafe character (%s)", u)
		}

		uri = append(uri, byte(b))
	}

	return string(uri), nil
}
