package constructs

import (
	"fmt"
	"slices"
	"time"

	"github.com/tony-montemuro/http/internal/lws"
)

const (
	ByteSeparator = '/'
	ByteParam     = ';'
	ByteQuery     = '?'
	Crlf          = "\r\n"
)

type Hex byte

func (b Hex) Value() (byte, error) {
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

type HttpByte byte

func (b HttpByte) IsEscape() bool {
	return b == '%'
}

func (b HttpByte) IsPChar() bool {
	return b.IsUnreserved() || slices.Contains([]HttpByte{':', '@', '&', '=', '.'}, b)
}

func (b HttpByte) IsUnreserved() bool {
	return b.IsAlpha() || b.IsNumeric() || b.IsSafe() || b.IsExtra() || (!b.IsReserved() && !b.IsUnsafe())
}

func (b HttpByte) IsAlpha() bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func (b HttpByte) IsNumeric() bool {
	return b >= '0' && b <= '9'
}

func (b HttpByte) IsHex() bool {
	_, err := Hex(b).Value()
	return err == nil
}

func (b HttpByte) IsReserved() bool {
	reserved := []HttpByte{';', '/', '?', ':', '@', '&', '=', '+'}
	return slices.Contains(reserved, b)
}

func (b HttpByte) IsSafe() bool {
	safe := []HttpByte{'$', '-', '_', '.'}
	return slices.Contains(safe, b)
}

func (b HttpByte) IsExtra() bool {
	extra := []HttpByte{'!', '*', '\'', '(', ')', ','}
	return slices.Contains(extra, b)
}

func (b HttpByte) IsControl() bool {
	return b < 32 || b == 127
}

func (b HttpByte) IsUnsafe() bool {
	unsafe := []HttpByte{' ', '"', '#', '%', '<', '>'}
	return b.IsControl() || slices.Contains(unsafe, b)
}

func (b HttpByte) IsUSAscii() bool {
	return b < 128
}

func (b HttpByte) IsQdTextByte() bool {
	return b.IsUSAscii() && !b.IsControl() && b != '"'
}

func (b HttpByte) IsTSpecial() bool {
	tSpecials := []HttpByte{'(', ')', '<', '>', '@', ',', ';', ':', '\\', '"', '/', '[', ']', '?', '=', ' ', '\t'}
	return slices.Contains(tSpecials, b)
}

type Token string

func (t Token) Validate() error {
	if len(t) == 0 {
		return fmt.Errorf("token cannot be empty")
	}

	for _, c := range t {
		if HttpByte(c).IsControl() {
			return fmt.Errorf("token cannot contain control character (%s)", t)
		}
		if !HttpByte(c).IsUSAscii() {
			return fmt.Errorf("token cannot contain extended ascii characters (%s)", t)
		}
		if HttpByte(c).IsTSpecial() {
			return fmt.Errorf("token contains invalid symbol (%s)", t)
		}
	}

	return nil
}

type Text string

func (t Text) Validate() error {
	i := 0

	for i < len(t) {
		isLws, next := lws.Check(string(t), i)
		if isLws {
			i = next
			continue
		}

		if HttpByte(t[i]).IsControl() {
			return fmt.Errorf("not a valid sequence of text bytes")
		}

		i++
	}

	return nil
}

type Date string

func (d Date) Parse() (time.Time, error) {
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

type QuotedString string

func (qs QuotedString) validate() error {
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

		c := HttpByte(qs[i])
		if !c.IsQdTextByte() {
			return fmt.Errorf("quoted string contains invalid character (%s)", qs)
		}
		i++
	}

	return nil
}

func (qs QuotedString) Parse() (string, error) {
	err := qs.validate()
	if err != nil {
		return string(qs), fmt.Errorf("not a quoted string (%s)", qs)
	}

	return string(qs[1 : len(qs)-1]), nil
}

type Word string

func (w Word) Validate() error {
	err := Token(w).Validate()
	if err == nil {
		return nil
	}

	err = QuotedString(w).validate()
	if err == nil {
		return nil
	}

	return fmt.Errorf("word is not a token or quoted string (%s)", w)
}

func (w Word) Parse() (string, error) {
	err := Token(w).Validate()
	if err == nil {
		return string(w), nil
	}

	s, err := QuotedString(w).Parse()
	if err == nil {
		return s, nil
	}

	return "", fmt.Errorf("not a word (%s)", w)
}

type Comment string

func (c Comment) Validate() error {
	if len(c) < 2 {
		return fmt.Errorf("comment is incomplete (%s)", c)
	}

	if c[0] != '(' {
		return fmt.Errorf("comment must begin with open parenthesis (%s)", c)
	}

	err := Text(c).Validate()
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
