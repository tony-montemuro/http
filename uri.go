package http

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tony-montemuro/http/internal/constructs"
)

type Uri interface {
	GetPath() []byte
	marshal() []byte
}

func unescapeSequence(data []byte, i int) (byte, error) {
	var b byte

	for j := 1; j <= 2; j++ {
		if i+j == len(data) {
			return b, ClientError{message: fmt.Sprintf("truncated escape sequence: (char pos: %d, \"%s\")", i+j-1, data)}
		}

		val, err := constructs.Hex(data[i+j]).Value()
		if err != nil {
			return b, ClientError{message: fmt.Sprintf("malformed escape sequence: (char pos: %d, \"%s\")", i+j, data[:i+j])}
		}

		b += (val) << (4 * (2 - j))
	}

	return b, nil
}

func parseUri(data []byte) (Uri, error) {
	var uri Uri
	var err error
	doesStartWithSchema := validateStartsWithScheme(data) == nil

	if doesStartWithSchema {
		uri, err = parseAbsoluteUri(data)
	} else {
		uri, err = parseRelativeUri(data)
	}

	return uri, err
}

func validateStartsWithScheme(data []byte) error {
	colonIndex := bytes.Index(data, []byte{':'})
	if colonIndex == -1 {
		return errors.New("could not determine schema")
	}

	err := constructs.ValidateScheme(string(data[:colonIndex]))
	return err
}

type AbsoluteUri struct {
	Scheme []byte
	Path   []byte
}

func (u AbsoluteUri) GetPath() []byte {
	return u.marshal()
}

func parseAbsoluteUri(data []byte) (AbsoluteUri, error) {
	var uri AbsoluteUri

	err := validateStartsWithScheme(data)
	if err != nil {
		return uri, err
	}

	scheme, remaining, _ := bytes.Cut(data, []byte{':'})
	uri.Scheme = scheme

	var path []byte
	i := 0

	for i < len(remaining) {
		b := constructs.HttpByte(remaining[i])

		if b.IsEscape() {
			c, err := unescapeSequence(remaining, i)
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

func parseRelativeUri(data []byte) (RelativeUri, error) {
	uri := RelativeUri{}
	start := 0

	if len(data) >= 2 && data[0] == constructs.ByteSeparator && data[1] == constructs.ByteSeparator {
		i := 2

		for i < len(data) && (constructs.HttpByte(data[i]).IsPChar() || data[i] == ';' || data[i] == '?') {
			i++
		}

		uri.NetLoc = data[2:i]
		start = i
	}

	if start == len(data) {
		return uri, nil
	}

	var err error
	var path, query []byte
	var params [][]byte

	if start > 0 || data[start] == constructs.ByteSeparator {
		path, params, query, err = parseAbsUri(data[start:])
	} else {
		path, params, query, err = parseRelPathUri(data[start:])
	}

	if err != nil {
		return uri, err
	}

	uri.Path = path
	uri.Params = params
	uri.Query = query

	return uri, nil
}

func parseAbsUri(data []byte) ([]byte, [][]byte, []byte, error) {
	var path, query []byte
	var params [][]byte
	var err error = fmt.Errorf("abs_path must begin with /")

	if len(data) == 0 || data[0] != constructs.ByteSeparator {
		return path, params, query, err
	}

	path, params, query, err = parseRelPathUri(data[1:])
	return append([]byte("/"), path...), params, query, err
}

func parseRelPathUri(data []byte) ([]byte, [][]byte, []byte, error) {
	var path, query []byte
	var params [][]byte

	paramsIndex := bytes.IndexByte(data, constructs.ByteParam)
	queryIndex := bytes.IndexByte(data, constructs.ByteQuery)

	var paramsSlice []byte
	var querySlice []byte

	if queryIndex != -1 {
		querySlice = data[queryIndex+1:]
	} else {
		queryIndex = len(data)
	}

	if paramsIndex != -1 && paramsIndex < queryIndex {
		paramsSlice = data[paramsIndex+1 : queryIndex]
	} else {
		paramsIndex = queryIndex
	}

	path, err := parseUriPath(data[:paramsIndex])
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri path: %s", err)}
	}

	params, err = parseUriParams(paramsSlice)
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri param(s): %s", err)}
	}

	query, err = parseUriQuery(querySlice)
	if err != nil {
		return path, params, query, ClientError{message: fmt.Sprintf("Invalid request uri querie(s): %s", err)}
	}

	return path, params, query, nil
}

func parseUriPath(data []byte) ([]byte, error) {
	var path [][]byte
	var res []byte
	unescaped := bytes.Split(data, []byte{byte(constructs.ByteSeparator)})

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
				c, err := unescapeSequence(p, j)
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

func parseUriParams(data []byte) ([][]byte, error) {
	var params [][]byte
	if len(data) == 0 {
		return params, nil
	}

	for p := range bytes.SplitSeq(data, []byte{byte(constructs.ByteParam)}) {
		j := 0
		var param []byte

		for j < len(p) {
			b := constructs.HttpByte(p[j])

			if b.IsEscape() {
				c, err := unescapeSequence(p, j)
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

func parseUriQuery(data []byte) ([]byte, error) {
	var query []byte
	i := 0

	for i < len(data) {
		b := constructs.HttpByte(data[i])

		if b.IsEscape() {
			c, err := unescapeSequence(data, i)
			if err != nil {
				return query, err
			}
			i += 3
			b = constructs.HttpByte(c)
		} else {
			i++
		}

		if !b.IsReserved() && !b.IsUnreserved() {
			return query, fmt.Errorf("queries contain invalid byte (%s)", data)
		}

		query = append(query, byte(b))
	}

	return query, nil
}
