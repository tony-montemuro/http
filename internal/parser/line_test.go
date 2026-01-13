package parser

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestRequestLineParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		line        []byte
		expected    ParsedRequestLine
		expectError bool
	}{
		{
			name:        "Standard GET method",
			line:        []byte("GET / HTTP/1.0"),
			expected:    ParsedRequestLine{Method: []byte("GET"), Uri: AbsPathUri{Path: [][]byte{{}}, Params: [][]byte{{}}, Query: []byte{}}, Version: []byte("1.0")},
			expectError: false,
		},
		{
			name:        "More complex POST method",
			line:        []byte("POST /data/document/4;param/3;test!true?foo=bar HTTP/2.0"),
			expected:    ParsedRequestLine{Method: []byte("POST"), Uri: AbsPathUri{Path: [][]byte{[]byte("data"), []byte("document"), []byte("4")}, Params: [][]byte{[]byte("param/3"), []byte("test!true")}, Query: []byte("foo=bar")}, Version: []byte("2.0")},
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
			res, err := requestLineParser(tt.line).Parse()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Error("did not get expected error!")
				return
			}

			assert.SliceEqual(t, res.Method, tt.expected.Method)
			assert.MatrixEqual(t, res.Uri.Path, tt.expected.Uri.Path)
			assert.MatrixEqual(t, res.Uri.Params, tt.expected.Uri.Params)
			assert.SliceEqual(t, res.Uri.Query, tt.expected.Uri.Query)
			assert.SliceEqual(t, res.Version, tt.expected.Version)
		})
	}
}

func TestVersionParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		version     []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "HTTP/1.0",
			version:     []byte("HTTP/1.0"),
			expected:    []byte("1.0"),
			expectError: false,
		},
		{
			name:        "HTTP/1.1",
			version:     []byte("HTTP/1.1"),
			expected:    []byte("1.1"),
			expectError: false,
		},
		{
			name:        "HTTP/2.0",
			version:     []byte("HTTP/2.0"),
			expected:    []byte("2.0"),
			expectError: false,
		},
		{
			name:        "Incomplete version",
			version:     []byte("HTTP"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (missing first digit)",
			version:     []byte("HTTP/.12"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (missing second digit)",
			version:     []byte("HTTP/34."),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (multiple decimal places)",
			version:     []byte("HTTP/1.2.3"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (non-numeric first number)",
			version:     []byte("HTTP/1f.0"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (non-numeric second number)",
			version:     []byte("HTTP/1.1e2"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Malformed version number (below 1.0)",
			version:     []byte("HTTP/0.9"),
			expected:    []byte(""),
			expectError: true,
		},
		{
			name:        "Wrong protocol",
			version:     []byte("REST/1.0"),
			expected:    []byte(""),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := versionParser(tt.version).parse()

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

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}
