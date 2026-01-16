package message

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/tony-montemuro/http/internal/constructs"
)

type requestLineParser []byte

func (rl requestLineParser) parse() (RequestLine, error) {
	parts := bytes.Split(rl, []byte(" "))
	if len(parts) != 3 {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: malformed request line (%s)", string(rl))}
	}

	m := Method(parts[0])
	err := m.Validate()
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with request method (%s)", err.Error())}
	}

	uri, err := absPathUriParser(parts[1]).parse()
	if err != nil {
		return RequestLine{}, err
	}

	version, err := versionParser(parts[2]).parse()
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with version (%s)", version)}
	}

	return RequestLine{Method: m, Uri: uri, Version: version}, nil
}

type versionParser string

func (v versionParser) parse() (string, error) {
	if len(v) < 8 {
		return string(v), fmt.Errorf("incomplete version (%s)", v)
	}

	data := strings.Split(string(v), string(constructs.ByteSeparator))
	if len(data) != 2 || !strings.Contains(data[1], ".") {
		return string(v), fmt.Errorf("could not determine version number (%s)", v)
	}

	if data[0] != "HTTP" {
		return string(v), fmt.Errorf("wrong protocol (%s)", data[0])
	}

	digits := strings.Split(data[1], ".")
	if len(digits) != 2 {
		return string(v), fmt.Errorf("malformed version number (%s)", data[1])
	}

	d1, err1 := strconv.Atoi(string(digits[0]))
	_, err2 := strconv.Atoi(string(digits[1]))
	if err1 != nil || err2 != nil {
		return string(v), fmt.Errorf("contains invalid characters (%s)", v)
	}
	if d1 == 0 {
		return string(v), fmt.Errorf("must be at least 1.0 (%s)", v)
	}

	return data[1], nil
}
