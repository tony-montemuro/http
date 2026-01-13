package lws

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

type checkResults struct {
	isLws    bool
	position int
}

type newLineResults struct {
	isNewLineLws bool
	min          int
	max          int
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		string   string
		position int
		expected checkResults
	}{
		{
			name:     "Single space",
			string:   " abc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 1,
			},
		},
		{
			name:     "Multiple space",
			string:   "    abc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 4,
			},
		},
		{
			name:     "Horizontal tab",
			string:   "\tabc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 1,
			},
		},
		{
			name:     "Mixed spaces and tabs",
			string:   " \t \tabc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 4,
			},
		},
		{
			name:     "CRLF followed by space",
			string:   "\r\n abc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 3,
			},
		},
		{
			name:     "CRLF followed by tab",
			string:   "\r\n\tabc",
			position: 0,
			expected: checkResults{
				isLws:    true,
				position: 3,
			},
		},
		{
			name:     "CRLF with no following space / tab",
			string:   "\r\nabc",
			position: 0,
			expected: checkResults{
				isLws:    false,
				position: 0,
			},
		},
		{
			name:     "Starting on CR without LF",
			string:   "\r abc",
			position: 0,
			expected: checkResults{
				isLws:    false,
				position: 0,
			},
		},
		{
			name:     "Starting on non-whitespace",
			string:   "Header: value",
			position: 0,
			expected: checkResults{
				isLws:    false,
				position: 0,
			},
		},
		{
			name:     "LWS in the middle of a string",
			string:   "abc\r\n\t def",
			position: 3,
			expected: checkResults{
				isLws:    true,
				position: 7,
			},
		},
		{
			name:     "LWS ending a string",
			string:   "abc \t\t ",
			position: 3,
			expected: checkResults{
				isLws:    true,
				position: 7,
			},
		},
		{
			name:     "CRLF with no following space / tab at the end of the string",
			string:   "abc\r\n",
			position: 3,
			expected: checkResults{
				isLws:    false,
				position: 3,
			},
		},
		{
			name:     "Position argument out of bounds",
			string:   "",
			position: 0,
			expected: checkResults{
				isLws:    false,
				position: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isLws, position := Check(tt.string, tt.position)
			assert.Equal(t, isLws, tt.expected.isLws)
			assert.Equal(t, position, tt.expected.position)
		})
	}
}

func TestNewLine(t *testing.T) {
	tests := []struct {
		name     string
		string   string
		position int
		expected newLineResults
	}{
		{
			name:     "Standard new line LWS",
			string:   "\r\n test",
			position: 0,
			expected: newLineResults{
				isNewLineLws: true,
				min:          3,
				max:          3,
			},
		},
		{
			name:     "Extended new line LWS",
			string:   "\r\n \t test",
			position: 0,
			expected: newLineResults{
				isNewLineLws: true,
				min:          3,
				max:          5,
			},
		},
		{
			name:     "New line LWS terminating at end of line",
			string:   "\r\n ",
			position: 0,
			expected: newLineResults{
				isNewLineLws: true,
				min:          3,
				max:          3,
			},
		},
		{
			name:     "Extended new line LWS terminating at end of line",
			string:   "test\r\n  \t ",
			position: 4,
			expected: newLineResults{
				isNewLineLws: true,
				min:          7,
				max:          10,
			},
		},
		{
			name:     "Non new line LWS",
			string:   "foo\t \r\n bar",
			position: 3,
			expected: newLineResults{
				isNewLineLws: false,
				min:          3,
				max:          3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isNewLineLws, lower, upper := NewLine(tt.string, tt.position)
			assert.Equal(t, isNewLineLws, tt.expected.isNewLineLws)
			assert.Equal(t, lower, tt.expected.min)
			assert.Equal(t, upper, tt.expected.max)
		})
	}
}

func TestTrimLeft(t *testing.T) {
	tests := []struct {
		name     string
		string   string
		expected string
	}{
		{
			name:     "Single space",
			string:   " abc",
			expected: "abc",
		},
		{
			name:     "Multiple spaces",
			string:   "   abc",
			expected: "abc",
		},
		{
			name:     "Single tab",
			string:   "\tabc",
			expected: "abc",
		},
		{
			name:     "Mixed spaces and tabs",
			string:   "\t  \t abc",
			expected: "abc",
		},
		{
			name:     "CRLF & space",
			string:   "\r\n abc",
			expected: "abc",
		},
		{
			name:     "CLRF & tab",
			string:   "\r\n\tabc",
			expected: "abc",
		},
		{
			name:     "Multiple LWS sequences at start",
			string:   " \r\n \t abc",
			expected: "abc",
		},
		{
			name:     "No LWS",
			string:   "abc",
			expected: "abc",
		},
		{
			name:     "CRLF without following SP/HT (not LWS)",
			string:   "\r\nabc",
			expected: "\r\nabc",
		},
		{
			name:     "Entire string is LWS",
			string:   "\r\n\t  \t ",
			expected: "",
		},
		{
			name:     "Empty string",
			string:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, TrimLeft(tt.string), tt.expected)
		})
	}
}

func TestTrimRight(t *testing.T) {
	tests := []struct {
		name     string
		string   string
		expected string
	}{
		{
			name:     "Single space",
			string:   "abc ",
			expected: "abc",
		},
		{
			name:     "Multiple spaces",
			string:   "abc    ",
			expected: "abc",
		},
		{
			name:     "Single tab",
			string:   "abc\t",
			expected: "abc",
		},
		{
			name:     "Mixed spaces and tabs",
			string:   "abc\t   \t ",
			expected: "abc",
		},
		{
			name:     "CRLF & space",
			string:   "abc \r\n ",
			expected: "abc",
		},
		{
			name:     "CLRF & tab",
			string:   "abc\t\r\n\t",
			expected: "abc",
		},
		{
			name:     "Multiple LWS sequences at end",
			string:   "abc \r\n \t ",
			expected: "abc",
		},
		{
			name:     "No LWS",
			string:   "abc",
			expected: "abc",
		},
		{
			name:     "CRLF without following SP/HT (not LWS)",
			string:   "abc\r\n",
			expected: "abc\r\n",
		},
		{
			name:     "Entire string is LWS",
			string:   "\r\n\t  \t ",
			expected: "",
		},
		{
			name:     "Scatted LWS",
			string:   "\r\n te\ts t\t  \r\n \t \r\n ",
			expected: "\r\n te\ts t",
		},
		{
			name:     "Empty string",
			string:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, TrimRight(tt.string), tt.expected)
		})
	}
}

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		string   string
		expected string
	}{
		{
			name:     "Single spaces",
			string:   " abc ",
			expected: "abc",
		},
		{
			name:     "Multiple LWS both sides",
			string:   "\r\n\t abc \t\r\n ",
			expected: "abc",
		},
		{
			name:     "Only left has LWS",
			string:   " \t abc",
			expected: "abc",
		},
		{
			name:     "Only right has LWS",
			string:   "abc \r\n ",
			expected: "abc",
		},
		{
			name:     "Entire string is LWS",
			string:   " \r\n \t ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Trim(tt.string), tt.expected)
		})
	}
}
