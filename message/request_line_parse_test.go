package message

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestRequestLineParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		line        []byte
		expected    RequestLine
		expectError bool
	}{
		{
			name:        "Standard GET method",
			line:        []byte("GET / HTTP/1.0"),
			expected:    RequestLine{Method: Method("GET"), Uri: AbsPathUri{Path: [][]byte{{}}, Params: [][]byte{{}}, Query: []byte{}}, Version: string("1.0")},
			expectError: false,
		},
		{
			name:        "More complex POST method",
			line:        []byte("POST /data/document/4;param/3;test!true?foo=bar HTTP/2.0"),
			expected:    RequestLine{Method: Method("POST"), Uri: AbsPathUri{Path: [][]byte{[]byte("data"), []byte("document"), []byte("4")}, Params: [][]byte{[]byte("param/3"), []byte("test!true")}, Query: []byte("foo=bar")}, Version: string("2.0")},
			expectError: false,
		},
		{
			name:        "Incomplete line",
			line:        []byte("GET /test"),
			expectError: true,
		},
		{
			name:        "Overcomplete line",
			line:        []byte("HEAD /test/document?baz=x HTTP/1.0 bad"),
			expectError: true,
		},
		{
			name:        "Bad method",
			line:        []byte("WR\rONG / HTTP/1.0"),
			expectError: true,
		},
		{
			name:        "Bad uri",
			line:        []byte("HEAD /malformed/u\nrl HTTP/1.0"),
			expectError: true,
		},
		{
			name:        "Bad version",
			line:        []byte("POST / HTTP/0.9"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := requestLineParser(tt.line).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res.Method, tt.expected.Method)
			assert.MatrixEqual(t, res.Uri.Path, tt.expected.Uri.Path)
			assert.MatrixEqual(t, res.Uri.Params, tt.expected.Uri.Params)
			assert.SliceEqual(t, res.Uri.Query, tt.expected.Uri.Query)
			assert.Equal(t, res.Version, tt.expected.Version)
		})
	}
}

func TestVersionParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		version     []byte
		expected    string
		expectError bool
	}{
		{
			name:        "HTTP/1.0",
			version:     []byte("HTTP/1.0"),
			expected:    "1.0",
			expectError: false,
		},
		{
			name:        "HTTP/1.1",
			version:     []byte("HTTP/1.1"),
			expected:    "1.1",
			expectError: false,
		},
		{
			name:        "HTTP/2.0",
			version:     []byte("HTTP/2.0"),
			expected:    "2.0",
			expectError: false,
		},
		{
			name:        "Incomplete version",
			version:     []byte("HTTP"),
			expectError: true,
		},
		{
			name:        "Malformed version number (missing first digit)",
			version:     []byte("HTTP/.12"),
			expectError: true,
		},
		{
			name:        "Malformed version number (missing second digit)",
			version:     []byte("HTTP/34."),
			expectError: true,
		},
		{
			name:        "Malformed version number (multiple decimal places)",
			version:     []byte("HTTP/1.2.3"),
			expectError: true,
		},
		{
			name:        "Malformed version number (non-numeric first number)",
			version:     []byte("HTTP/1f.0"),
			expectError: true,
		},
		{
			name:        "Malformed version number (non-numeric second number)",
			version:     []byte("HTTP/1.1e2"),
			expectError: true,
		},
		{
			name:        "Malformed version number (below 1.0)",
			version:     []byte("HTTP/0.9"),
			expectError: true,
		},
		{
			name:        "Wrong protocol",
			version:     []byte("REST/1.0"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := versionParser(tt.version).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res, tt.expected)
		})
	}
}
