package parser

import (
	"bytes"
	"fmt"
)

type escapeSequence []byte

func (s escapeSequence) unescape(i int) (byte, error) {
	var b byte

	for j := 1; j <= 2; j++ {
		if i+j == len(s) {
			return b, ClientError{message: fmt.Sprintf("truncated escape sequence: (char pos: %d, \"%s\")", i+j-1, s)}
		}

		val, err := hex(s[i+j]).value()
		if err != nil {
			return b, ClientError{message: fmt.Sprintf("malformed escape sequence: (char pos: %d, \"%s\")", i+j, s[:i+j])}
		}

		b += (val) << (4 * (2 - j))
	}

	return b, nil
}

type AbsPathUri struct {
	Path   [][]byte
	Params [][]byte
	Query  []byte
}

type absPathUriParser []byte

func (ap absPathUriParser) parse() (AbsPathUri, error) {
	uri := AbsPathUri{}
	if len(ap) == 0 {
		return uri, ClientError{message: "Invalid request uri: missing uri"}
	}

	if ap[0] != byteSeparator {
		return uri, ClientError{fmt.Sprintf("Invalid request uri: uri must begin with '%c'", byteSeparator)}
	}

	paramsIndex := bytes.IndexByte(ap, byteParam)
	queryIndex := bytes.IndexByte(ap, byteQuery)

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

	pathSlice := ap[1:paramsIndex]

	path, err := uriPathParser(pathSlice).parse()
	if err != nil {
		return uri, ClientError{message: fmt.Sprintf("Invalid request uri path: %s", err)}
	}
	uri.Path = path

	params, err := uriParamsParser(paramsSlice).parse()
	if err != nil {
		return uri, ClientError{message: fmt.Sprintf("Invalid request uri param(s): %s", err)}
	}
	uri.Params = params

	query, err := uriQueryParser(querySlice).parse()
	if err != nil {
		return uri, ClientError{message: fmt.Sprintf("Invalid request uri querie(s): %s", err)}
	}
	uri.Query = query

	return uri, nil
}

type uriPathParser []byte

func (p uriPathParser) parse() ([][]byte, error) {
	var path [][]byte
	unescaped := bytes.Split(p, []byte{byte(byteSeparator)})

	// special case: if we have at least 1 segment, the first segment cannot be empty according to RFC 1945 (see: https://datatracker.ietf.org/doc/html/rfc1945#section-3.2.1)
	if len(unescaped) > 1 && len(unescaped[0]) == 0 {
		return unescaped, fmt.Errorf("first segment cannot be empty")
	}

	for _, p := range unescaped {
		j := 0
		var part []byte

		for j < len(p) {
			b := httpByte(p[j])

			if b.isEscape() {
				c, err := escapeSequence(p).unescape(j)
				if err != nil {
					return path, err
				}
				j += 3
				b = httpByte(c)
			} else {
				j++
			}

			if !b.ispChar() {
				return path, fmt.Errorf("path contains invalid byte (%s)", p)
			}

			part = append(part, byte(b))
		}

		path = append(path, part)
	}

	return path, nil
}

type uriParamsParser []byte

func (p uriParamsParser) parse() ([][]byte, error) {
	var params [][]byte

	for p := range bytes.SplitSeq(p, []byte{byte(byteParam)}) {
		j := 0
		var param []byte

		for j < len(p) {
			b := httpByte(p[j])

			if b.isEscape() {
				c, err := escapeSequence(p).unescape(j)
				if err != nil {
					return params, err
				}
				j += 3
				b = httpByte(c)
			} else {
				j++
			}

			if !b.ispChar() && b != byteSeparator {
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
		b := httpByte(q[i])

		if b.isEscape() {
			c, err := escapeSequence(q).unescape(i)
			if err != nil {
				return query, err
			}
			i += 3
			b = httpByte(c)
		} else {
			i++
		}

		if !b.isReserved() && !b.isUnreserved() {
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
		b := httpByte(u[i])

		if b.isEscape() {
			c, err := escapeSequence(u).unescape(i)
			if err != nil {
				return string(u), err
			}
			i += 3
			b = httpByte(c)
		} else {
			i++
		}

		if b.isUnsafe() && b != '#' {
			return string(u), fmt.Errorf("uri contains at least 1 unsafe character (%s)", u)
		}

		uri = append(uri, byte(b))
	}

	return string(uri), nil
}
