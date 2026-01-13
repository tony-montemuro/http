package parser

import (
	"testing"
	"time"

	"github.com/tony-montemuro/http/internal/assert"
)

type byteCheck struct {
	name     string
	byte     byte
	expected bool
}

type validateCheck struct {
	name        string
	string      string
	expectError bool
}

type parseCheck struct {
	name        string
	string      string
	expected    string
	expectError bool
}

func TestHttpByte_isEscape(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Percent sign (%)",
			byte:     '%',
			expected: true,
		},
		{
			name:     "Not percent sign (%)",
			byte:     'a',
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isEscape(), tt.expected)
		})
	}
}

func TestHttpByte_isExtra(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Exclaimation mark (!)",
			byte:     '!',
			expected: true,
		},
		{
			name:     "Single quote (')",
			byte:     '\'',
			expected: true,
		},
		{
			name:     "A character (a)",
			byte:     'a',
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isExtra(), tt.expected)
		})
	}
}

func TestHttpByte_isUnsafe(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Byte 0",
			byte:     0,
			expected: true,
		},
		{
			name:     "Byte 31",
			byte:     31,
			expected: true,
		},
		{
			name:     "Byte 127",
			byte:     127,
			expected: true,
		},
		{
			name:     "Explicitly unsafe byte",
			byte:     '#',
			expected: true,
		},
		{
			name:     "Safe byte alpha",
			byte:     'a',
			expected: false,
		},
		{
			name:     "Safe byte numeric",
			byte:     '0',
			expected: false,
		},
		{
			name:     "Safe byte symbol",
			byte:     ':',
			expected: false,
		},
		{
			name:     "Safe high byte",
			byte:     255,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isUnsafe(), tt.expected)
		})
	}
}

func TestHttpByte_isSafe(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Dollar sign byte ($)",
			byte:     '$',
			expected: true,
		},
		{
			name:     "Underscore byte (_)",
			byte:     '_',
			expected: true,
		},
		{
			name:     "Alpha byte (A)",
			byte:     'A',
			expected: false,
		},
		{
			name:     "Numeric byte (1)",
			byte:     '1',
			expected: false,
		},
		{
			name:     "Unsafe symbol byte (/)",
			byte:     '/',
			expected: false,
		},
		{
			name:     "Control byte (1)",
			byte:     1,
			expected: false,
		},
		{
			name:     "High byte (254)",
			byte:     254,
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isSafe(), tt.expected)
		})
	}
}

func TestHttpByte_isReserved(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Semicolon byte (;)",
			byte:     ';',
			expected: true,
		},
		{
			name:     "Equal byte (=)",
			byte:     '=',
			expected: true,
		},
		{
			name:     "Alpha byte (b)",
			byte:     'b',
			expected: false,
		},
		{
			name:     "Numeric byte (2)",
			byte:     '2',
			expected: false,
		},
		{
			name:     "Unreserved symbol byte (/)",
			byte:     '-',
			expected: false,
		},
		{
			name:     "Control byte (2)",
			byte:     2,
			expected: false,
		},
		{
			name:     "High byte (253)",
			byte:     253,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isReserved(), tt.expected)
		})
	}
}

func TestHttpByte_isHex(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Numeric byte (0)",
			byte:     '0',
			expected: true,
		},
		{
			name:     "Numeric byte 2 (9)",
			byte:     '9',
			expected: true,
		},
		{
			name:     "Lower alpha byte (a)",
			byte:     'a',
			expected: true,
		},
		{
			name:     "Lower alpha byte 2 (f)",
			byte:     'f',
			expected: true,
		},
		{
			name:     "Lower alpha byte out-of-range (g)",
			byte:     'g',
			expected: false,
		},
		{
			name:     "Upper alpha byte (A)",
			byte:     'A',
			expected: true,
		},
		{
			name:     "Upper alpha byte 2 (F)",
			byte:     'F',
			expected: true,
		},
		{
			name:     "Upper alpha byte out-of-range (G)",
			byte:     'G',
			expected: false,
		},
		{
			name:     "Symbol byte (;)",
			byte:     ';',
			expected: false,
		},
		{
			name:     "Control byte (3)",
			byte:     3,
			expected: false,
		},
		{
			name:     "High byte (252)",
			byte:     252,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isHex(), tt.expected)
		})
	}
}

func TestHttpByte_isNumeric(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Numeric byte (0)",
			byte:     '0',
			expected: true,
		},
		{
			name:     "Numeric byte 2 (9)",
			byte:     '9',
			expected: true,
		},
		{
			name:     "Alpha byte (M)",
			byte:     'M',
			expected: false,
		},
		{
			name:     "Symbol byte (*)",
			byte:     '*',
			expected: false,
		},
		{
			name:     "Control byte (4)",
			byte:     4,
			expected: false,
		},
		{
			name:     "High byte (251)",
			byte:     251,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isNumeric(), tt.expected)
		})
	}
}

func TestHttpByte_isAlpha(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Lower alpha byte (a)",
			byte:     'a',
			expected: true,
		},
		{
			name:     "Lower alpha byte 2 (z)",
			byte:     'z',
			expected: true,
		},
		{
			name:     "Upper alpha byte (A)",
			byte:     'A',
			expected: true,
		},
		{
			name:     "Upper alpha byte 2 (Z)",
			byte:     'Z',
			expected: true,
		},
		{
			name:     "Symbol byte (+)",
			byte:     '+',
			expected: false,
		},
		{
			name:     "Control byte (5)",
			byte:     5,
			expected: false,
		},
		{
			name:     "High byte (250)",
			byte:     250,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isAlpha(), tt.expected)
		})
	}
}

func TestHttpByte_isUnreserved(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Lower alpha byte (F)",
			byte:     'F',
			expected: true,
		},
		{
			name:     "Upper alpha byte (Q)",
			byte:     'Q',
			expected: true,
		},
		{
			name:     "Numeric byte (6)",
			byte:     '6',
			expected: true,
		},
		{
			name:     "Safe byte (.)",
			byte:     '.',
			expected: true,
		},
		{
			name:     "Extra byte (()",
			byte:     '(',
			expected: true,
		},
		{
			name:     "Reserved byte (#)",
			byte:     '#',
			expected: false,
		},
		{
			name:     "Unsafe byte (>)",
			byte:     '>',
			expected: false,
		},
		{
			name:     "High byte (249)",
			byte:     249,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isUnreserved(), tt.expected)
		})
	}
}

func TestHttpByte_isIspChar(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Lower alpha byte (F)",
			byte:     'F',
			expected: true,
		},
		{
			name:     "Upper alpha byte (Q)",
			byte:     'Q',
			expected: true,
		},
		{
			name:     "Numeric byte (6)",
			byte:     '6',
			expected: true,
		},
		{
			name:     "Safe byte (.)",
			byte:     '.',
			expected: true,
		},
		{
			name:     "Extra byte (()",
			byte:     '(',
			expected: true,
		},
		{
			name:     "At sign (@)",
			byte:     '@',
			expected: true,
		},
		{
			name:     "Reserved byte (#)",
			byte:     '#',
			expected: false,
		},
		{
			name:     "Unsafe byte (>)",
			byte:     '>',
			expected: false,
		},
		{
			name:     "High byte (249)",
			byte:     249,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).ispChar(), tt.expected)
		})
	}
}

func TestHttpByte_isControl(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Last control byte (31)",
			byte:     31,
			expected: true,
		},
		{
			name:     "First non-control byte (32)",
			byte:     32,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isControl(), tt.expected)
		})
	}
}

func TestHttpByte_isUSAscii(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Last US ASCII byte (127)",
			byte:     127,
			expected: true,
		},
		{
			name:     "First non-US ASCII byte (128)",
			byte:     128,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isUSAscii(), tt.expected)
		})
	}
}

func TestHttpByte_isQdTextByte(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Alpha character (z)",
			byte:     'z',
			expected: true,
		},
		{
			name:     "Numeric character (8)",
			byte:     '8',
			expected: true,
		},
		{
			name:     "Valid symbol (-)",
			byte:     '-',
			expected: true,
		},
		{
			name:     "High byte (248)",
			byte:     248,
			expected: false,
		},
		{
			name:     "Double quote (\")",
			byte:     '"',
			expected: false,
		},
		{
			name:     "Control character (15)",
			byte:     15,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isQdTextByte(), tt.expected)
		})
	}
}

func TestHttpByte_isTSpecial(t *testing.T) {
	tests := []byteCheck{
		{
			name:     "Alpha character (z)",
			byte:     'z',
			expected: false,
		},
		{
			name:     "Numeric character (8)",
			byte:     '8',
			expected: false,
		},
		{
			name:     "Valid symbol (-)",
			byte:     '-',
			expected: false,
		},
		{
			name:     "High byte (248)",
			byte:     248,
			expected: false,
		},
		{
			name:     "Open parenthesis (()",
			byte:     '(',
			expected: true,
		},
		{
			name:     "Horizontal tab (\t)",
			byte:     '\t',
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, httpByte(tt.byte).isTSpecial(), tt.expected)
		})
	}
}

func TestToken_validate(t *testing.T) {
	tests := []validateCheck{
		{
			name:        "Standard token (abc!123)",
			string:      "abc!123",
			expectError: false,
		},
		{
			name:        "String with control characters (def\n456)",
			string:      "def\n456",
			expectError: true,
		},
		{
			name:        "String with extended ASCII characters (ghi[200]789)",
			string:      string([]byte{'g', 'h', 'i', 200, '7', '8', '9'}),
			expectError: true,
		},
		{
			name:        "String with TSpecial characters (jkl\\098)",
			string:      "jkl\\098",
			expectError: true,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := token(tt.string).validate()
			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
			}
		})
	}
}

func TestQuotedString_validate(t *testing.T) {
	tests := []validateCheck{
		{
			name:        "Standard quoted string (\"abc!123\")",
			string:      "\"abc!123\"",
			expectError: false,
		},
		{
			name:        "Quoted string with whitespace (\"d\t\t\tef \t45 6\")",
			string:      "\"d\t\t\tef \t45 6\"",
			expectError: false,
		},
		{
			name:        "Quoted string with trailing whitespace (\"foobar\t\t\t\t\t\t    \t \")",
			string:      "\"foobar\t\t\t\t\t\t    \t \"",
			expectError: false,
		},
		{
			name:        "Quote string with internal double quote (\"this is b\"ad!\")",
			string:      "\"this is b\"ad!\"",
			expectError: true,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: true,
		},
		{
			name:        "Single double quote (\")",
			string:      "\"",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := quotedString(tt.string).validate()
			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
			}
		})
	}
}

func TestQuotedString_parse(t *testing.T) {
	tests := []parseCheck{
		{
			name:        "Standard quoted string (\"abc!123\")",
			string:      "\"abc!123\"",
			expected:    "abc!123",
			expectError: false,
		},
		{
			name:        "Quoted string with whitespace (\"d\t\t\tef \t45 6\")",
			string:      "\"d\t\t\tef \t45 6\"",
			expected:    "d\t\t\tef \t45 6",
			expectError: false,
		},
		{
			name:        "Quoted string with trailing whitespace (\"foobar\t\t\t\t\t\t    \t \")",
			string:      "\"foobar\t\t\t\t\t\t    \t \"",
			expected:    "foobar\t\t\t\t\t\t    \t ",
			expectError: false,
		},
		{
			name:        "Quote string with internal double quote (\"this is b\"ad!\")",
			string:      "\"this is b\"ad!\"",
			expectError: true,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: true,
		},
		{
			name:        "Single double quote (\")",
			string:      "\"",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := quotedString(tt.string).parse()
			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s (res: %s)", err.Error(), res)
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! res: %s", res)
			}

			assert.Equal(t, res, tt.expected)
		})
	}
}

func TestWord_validate(t *testing.T) {
	tests := []validateCheck{
		{
			name:        "Standard token (abc!123)",
			string:      "abc!123",
			expectError: false,
		},
		{
			name:        "String with control characters (def\n456)",
			string:      "def\n456",
			expectError: true,
		},
		{
			name:        "String with extended ASCII characters (ghi[200]789)",
			string:      string([]byte{'g', 'h', 'i', 200, '7', '8', '9'}),
			expectError: true,
		},
		{
			name:        "String with TSpecial characters (jkl\\098)",
			string:      "jkl\\098",
			expectError: true,
		},
		{
			name:        "Standard quoted string (\"abc!123\")",
			string:      "\"abc!123\"",
			expectError: false,
		},
		{
			name:        "Quoted string with whitespace (\"d\t\t\tef \t45 6\")",
			string:      "\"d\t\t\tef \t45 6\"",
			expectError: false,
		},
		{
			name:        "Quoted string with trailing whitespace (\"foobar\t\t\t\t\t\t    \t \")",
			string:      "\"foobar\t\t\t\t\t\t    \t \"",
			expectError: false,
		},
		{
			name:        "Quote string with internal double quote (\"this is b\"ad!\")",
			string:      "\"this is b\"ad!\"",
			expectError: true,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: true,
		},
		{
			name:        "Single double quote (\")",
			string:      "\"",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := word(tt.string).validate()
			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
			}
		})
	}
}

func TestWord_parse(t *testing.T) {
	tests := []parseCheck{
		{
			name:        "Standard token (abc!123)",
			string:      "abc!123",
			expected:    "abc!123",
			expectError: false,
		},
		{
			name:        "String with control characters (def\n456)",
			string:      "def\n456",
			expectError: true,
		},
		{
			name:        "String with extended ASCII characters (ghi[200]789)",
			string:      string([]byte{'g', 'h', 'i', 200, '7', '8', '9'}),
			expectError: true,
		},
		{
			name:        "String with TSpecial characters (jkl\\098)",
			string:      "jkl\\098",
			expectError: true,
		},
		{
			name:        "Standard quoted string (\"abc!123\")",
			string:      "\"abc!123\"",
			expected:    "abc!123",
			expectError: false,
		},
		{
			name:        "Quoted string with whitespace (\"d\t\t\tef \t45 6\")",
			string:      "\"d\t\t\tef \t45 6\"",
			expected:    "d\t\t\tef \t45 6",
			expectError: false,
		},
		{
			name:        "Quoted string with trailing whitespace (\"foobar\t\t\t\t\t\t    \t \")",
			string:      "\"foobar\t\t\t\t\t\t    \t \"",
			expected:    "foobar\t\t\t\t\t\t    \t ",
			expectError: false,
		},
		{
			name:        "Quote string with internal double quote (\"this is b\"ad!\")",
			string:      "\"this is b\"ad!\"",
			expectError: true,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: true,
		},
		{
			name:        "Single double quote (\")",
			string:      "\"",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := word(tt.string).parse()
			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s (res: %s)", err.Error(), res)
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! res: %s", res)
			}
		})
	}
}

func TestHex_value(t *testing.T) {
	tests := []struct {
		name        string
		byte        byte
		expected    byte
		expectError bool
	}{
		{
			name:        "Numeric byte (0)",
			byte:        '0',
			expected:    0,
			expectError: false,
		},
		{
			name:        "Lower alpha byte (f)",
			byte:        'f',
			expected:    15,
			expectError: false,
		},
		{
			name:        "Upper alpha byte (F)",
			byte:        'F',
			expected:    15,
			expectError: false,
		},
		{
			name:        "Invalid byte (Z)",
			byte:        'Z',
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := hex(tt.byte).value()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
			}

			assert.Equal(t, res, tt.expected)
		})
	}
}

func TestText_validate(t *testing.T) {
	tests := []validateCheck{
		{
			name:        "Generic text",
			string:      "Foo bar baz",
			expectError: false,
		},
		{
			name:        "Text containing LWS",
			string:      "Foo\r\n\t Bar\r\n\t \r\n \t\t ",
			expectError: false,
		},
		{
			name:        "Text containing control characters",
			string:      "Foo\004Bar",
			expectError: true,
		},
		{
			name:        "Text containing extended ASCII",
			string:      string([]byte{150, 200, 175, 255, 'a', ' ', 't'}),
			expectError: false,
		},
		{
			name:        "Text containing exclusively LWS",
			string:      "   \t\t  \r\n\t    \t  \t\t\t\t \r\n    \r\n \t",
			expectError: false,
		},
		{
			name:        "Empty string",
			string:      "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := text(tt.string).validate()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
			}
		})
	}
}

func TestDateParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		dateVal     string
		expected    time.Time
		expectError bool
	}{
		{
			name:        "Standard RFC 1123",
			dateVal:     "Sun, 06 Nov 1994 08:49:37 GMT",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: false,
		},
		{
			name:        "Standard RFC 850",
			dateVal:     "Sunday, 06-Nov-94 08:49:37 GMT",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: false,
		},
		{
			name:        "Standard asctime",
			dateVal:     "Sun Nov  6 08:49:37 1994",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: false,
		},
		{
			name:        "Asctime with two digit day",
			dateVal:     "Sun Nov 06 08:49:37 1994",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: false,
		},
		{
			name:        "Non-GMT RFC 1123",
			dateVal:     "Sun, 06 Nov 1994 08:49:37 PST",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: true,
		},
		{
			name:        "Non-GMT RFC 850",
			dateVal:     "Sunday, 06-Nov-94 08:49:37 CST",
			expected:    time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := dateParser(tt.dateVal).parse()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s (%v)", err.Error(), res)
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! result: %s", res)
				return
			}

			assert.DateEqual(t, res, tt.expected)
		})
	}
}

func TestProductTokenParser_parse(t *testing.T) {
	tests := []struct {
		name         string
		productToken string
		expected     parsedProductToken
		expectError  bool
	}{
		{
			name:         "Standard product token",
			productToken: "Apache",
			expected: parsedProductToken{
				Product: "Apache",
				Version: "",
			},
			expectError: false,
		},
		{
			name:         "More complex product token",
			productToken: "libwww/2.17be",
			expected: parsedProductToken{
				Product: "libwww",
				Version: "2.17be",
			},
			expectError: false,
		},
		{
			name:         "Product token with valid punctuation",
			productToken: "lib_http-client/1.0.0-beta_2",
			expected: parsedProductToken{
				Product: "lib_http-client",
				Version: "1.0.0-beta_2",
			},
		},
		{
			name:         "Product token with multiple forward slashes",
			productToken: "Apache/2.0/Test",
			expectError:  true,
		},
		{
			name:         "Product token containing invalid characters",
			productToken: "go/te[@]st",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := productTokenParser(tt.productToken).parse()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s (%v)", err.Error(), res)
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! result: %s", res)
				return
			}

			assert.Equal(t, res.Product, tt.expected.Product)
			assert.Equal(t, res.Version, tt.expected.Version)
		})
	}
}

func TestComment_validate(t *testing.T) {
	tests := []validateCheck{
		{
			name:        "Empty comment",
			string:      "()",
			expectError: false,
		},
		{
			name:        "Simple text comment",
			string:      "(hello world)",
			expectError: false,
		},
		{
			name:        "Comment with punctuation",
			string:      "(version 1.2.3-alpha!)",
			expectError: false,
		},
		{
			name:        "Nested comment",
			string:      "(foo (bar) baz)",
			expectError: false,
		},
		{
			name:        "Deeply nested comment",
			string:      "(a (b (c (d))) e)",
			expectError: false,
		},
		{
			name:        "Adjacent nested comments",
			string:      "(a (b)(c)(d) e)",
			expectError: false,
		},
		{
			name:        "Unbalanced opening parenthesis",
			string:      "(foo",
			expectError: true,
		},
		{
			name:        "No starting open parenthesis",
			string:      "foo)",
			expectError: true,
		},
		{
			name:        "Unbalanced closing parenthesis",
			string:      "(foo (bar)))",
			expectError: true,
		},
		{
			name:        "Illegal character inside comment",
			string:      "(foo \x01 bar)",
			expectError: true,
		},
		{
			name:        "Incomplete comment",
			string:      "(",
			expectError: true,
		},
	}

	for _, tt := range tests {
		err := comment(tt.string).validate()

		if err != nil {
			if !tt.expectError {
				t.Errorf("got unexpected error: %s", err.Error())
			}
			return
		}

		if tt.expectError {
			t.Error("did not get expected error!")
		}
	}
}
