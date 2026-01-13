package parser

import (
	"bytes"
	"fmt"
	"strconv"
)

type ParsedRequestLine struct {
	Method  []byte
	Uri     AbsPathUri
	Version []byte
}

type requestLineParser []byte

func (rl requestLineParser) Parse() (ParsedRequestLine, error) {
	parts := bytes.Split(rl, []byte(" "))
	if len(parts) != 3 {
		return ParsedRequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: malformed request line (%s)", string(rl))}
	}

	m := parts[0]
	err := token(m).validate()
	if err != nil {
		return ParsedRequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with request method (%s)", err.Error())}
	}

	uri, err := absPathUriParser(parts[1]).parse()
	if err != nil {
		return ParsedRequestLine{}, err
	}

	version, err := versionParser(parts[2]).parse()
	if err != nil {
		return ParsedRequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with version (%s)", version)}
	}

	return ParsedRequestLine{Method: m, Uri: uri, Version: version}, nil
}

type versionParser []byte

func (v versionParser) parse() ([]byte, error) {
	if len(v) < 8 {
		return v, fmt.Errorf("incomplete version (%s)", v)
	}

	data := bytes.Split(v, []byte{byteSeparator})
	if len(data) != 2 || !bytes.Contains(data[1], []byte{'.'}) {
		return v, fmt.Errorf("could not determine version number (%s)", v)
	}

	if !bytes.Equal(data[0], []byte("HTTP")) {
		return v, fmt.Errorf("wrong protocol (%s)", data[0])
	}

	digits := bytes.Split(data[1], []byte{'.'})
	if len(digits) != 2 {
		return v, fmt.Errorf("malformed version number (%s)", data[1])
	}

	d1, err1 := strconv.Atoi(string(digits[0]))
	_, err2 := strconv.Atoi(string(digits[1]))
	if err1 != nil || err2 != nil {
		return v, fmt.Errorf("contains invalid characters (%s)", v)
	}
	if d1 == 0 {
		return v, fmt.Errorf("must be at least 1.0 (%s)", v)
	}

	return data[1], nil
}
