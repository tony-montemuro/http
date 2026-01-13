package parser

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tony-montemuro/http/internal/lws"
)

const (
	byteSeparator = '/'
	byteParam     = ';'
	byteQuery     = '?'
	crlf          = "\r\n"
)

type httpByte byte

func (b httpByte) isEscape() bool {
	return b == '%'
}

func (b httpByte) ispChar() bool {
	return b.isUnreserved() || slices.Contains([]httpByte{':', '@', '&', '=', '.'}, b)
}

func (b httpByte) isUnreserved() bool {
	return b.isAlpha() || b.isNumeric() || b.isSafe() || b.isExtra() || (!b.isReserved() && !b.isUnsafe())
}

func (b httpByte) isAlpha() bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func (b httpByte) isNumeric() bool {
	return b >= '0' && b <= '9'
}

func (b httpByte) isHex() bool {
	_, err := hex(b).value()
	return err == nil
}

func (b httpByte) isReserved() bool {
	reserved := []httpByte{';', '/', '?', ':', '@', '&', '=', '+'}
	return slices.Contains(reserved, b)
}

func (b httpByte) isSafe() bool {
	safe := []httpByte{'$', '-', '_', '.'}
	return slices.Contains(safe, b)
}

func (b httpByte) isExtra() bool {
	extra := []httpByte{'!', '*', '\'', '(', ')', ','}
	return slices.Contains(extra, b)
}

func (b httpByte) isControl() bool {
	return b < 32 || b == 127
}

func (b httpByte) isUnsafe() bool {
	unsafe := []httpByte{' ', '"', '#', '%', '<', '>'}
	return b.isControl() || slices.Contains(unsafe, b)
}

func (b httpByte) isUSAscii() bool {
	return b < 128
}

func (b httpByte) isQdTextByte() bool {
	return b.isUSAscii() && !b.isControl() && b != '"'
}

func (b httpByte) isTSpecial() bool {
	tSpecials := []httpByte{'(', ')', '<', '>', '@', ',', ';', ':', '\\', '"', '/', '[', ']', '?', '=', ' ', '\t'}
	return slices.Contains(tSpecials, b)
}

type token string

func (t token) validate() error {
	if len(t) == 0 {
		return fmt.Errorf("token cannot be empty")
	}

	for _, c := range t {
		if httpByte(c).isControl() {
			return fmt.Errorf("token cannot contain control character (%s)", t)
		}
		if !httpByte(c).isUSAscii() {
			return fmt.Errorf("token cannot contain extended ascii characters (%s)", t)
		}
		if httpByte(c).isTSpecial() {
			return fmt.Errorf("token contains invalid symbol (%s)", t)
		}
	}

	return nil
}

type quotedString string

func (qs quotedString) validate() error {
	if len(qs) < 2 {
		return fmt.Errorf("incomplete quote string (%s)", qs)
	}

	if qs[0] != '"' || qs[len(qs)-1] != '"' {
		return fmt.Errorf("quoted string must begin and end with a \" character (%s)", qs)
	}

	i := 1
	for i < len(qs)-1 {
		isLws, next := lws.Check(string(qs), i)
		if isLws {
			i = next
			continue
		}

		c := httpByte(qs[i])
		if !c.isQdTextByte() {
			return fmt.Errorf("quoted string contains invalid character (%s)", qs)
		}
		i++
	}

	return nil
}

func (qs quotedString) parse() (string, error) {
	err := qs.validate()
	if err != nil {
		return string(qs), fmt.Errorf("not a quoted string (%s)", qs)
	}

	return string(qs[1 : len(qs)-1]), nil
}

type word string

func (w word) validate() error {
	err := token(w).validate()
	if err == nil {
		return nil
	}

	err = quotedString(w).validate()
	if err == nil {
		return nil
	}

	return fmt.Errorf("word is not a token or quoted string (%s)", w)
}

func (w word) parse() (string, error) {
	err := token(w).validate()
	if err == nil {
		return string(w), nil
	}

	s, err := quotedString(w).parse()
	if err == nil {
		return s, nil
	}

	return "", fmt.Errorf("not a word (%s)", w)
}

type hex byte

func (b hex) value() (byte, error) {
	switch {
	case b >= '0' && b <= '9':
		return byte(b - '0'), nil
	case b >= 'a' && b <= 'f':
		return byte(b - 'a' + 10), nil
	case b >= 'A' && b <= 'F':
		return byte(b - 'A' + 10), nil
	}

	return 0, fmt.Errorf("escape sequence contains non-hex byte")
}

type text string

func (t text) validate() error {
	i := 0

	for i < len(t) {
		isLws, next := lws.Check(string(t), i)
		if isLws {
			i = next
			continue
		}

		if httpByte(t[i]).isControl() {
			return fmt.Errorf("not a valid sequence of text bytes")
		}

		i++
	}

	return nil
}

type dateParser string

func (d dateParser) parse() (time.Time, error) {
	var date time.Time

	res, err := time.Parse(time.RFC850, string(d))
	if err == nil {
		date = res
	}

	res, err = time.Parse(time.RFC1123, string(d))
	if err == nil {
		date = res
	}

	res, err = time.Parse(time.ANSIC, string(d))
	if err == nil {
		date = res.In(time.FixedZone("GMT", 0))
	}

	if date.IsZero() {
		return date, fmt.Errorf("could not parse date: %s", d)
	}

	tz, _ := date.Zone()
	if tz != "GMT" {
		return date, fmt.Errorf("timezone must be GMT: %s", d)
	}

	return date, nil

}

type productTokenParser string

func (p productTokenParser) parse() (parsedProductToken, error) {
	product := parsedProductToken{}
	parts := strings.Split(string(p), "/")
	if len(parts) > 2 {
		return product, fmt.Errorf("product token can only contain up to 1 forward slash (%s)", p)
	}

	err := token(parts[0]).validate()
	if err != nil {
		return product, fmt.Errorf("invalid product token (%s)", p)
	}
	product.Product = parts[0]

	if len(parts) == 2 {
		err := token(parts[1]).validate()
		if err != nil {
			return product, fmt.Errorf("invalid product token (%s)", p)
		}
		product.Version = parts[1]
	}

	return product, nil
}

type comment string

func (c comment) validate() error {
	if len(c) < 2 {
		return fmt.Errorf("comment is incomplete (%s)", c)
	}

	if c[0] != '(' {
		return fmt.Errorf("comment must begin with open parenthesis (%s)", c)
	}

	err := text(c).validate()
	if err != nil {
		return fmt.Errorf("comment contains invalid bytes (%s)", c)
	}

	score := 0
	for _, val := range c {
		if val == '(' {
			score++
		}
		if val == ')' {
			score--
		}
		if score < 0 {
			return fmt.Errorf("malformed comment (%s)", c)
		}
	}

	if score > 0 {
		return fmt.Errorf("comment not properly closed (%s)", c)
	}

	return nil
}
