package parser

import (
	"bytes"
	"compress/lzw"
	"encoding/base64"
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestGzipDecode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []byte
		expectError bool
	}{
		{
			name:        "Hello, World!",
			input:       "H4sIAAAAAAAAA/JIzcnJ11EIzy/KSVEEAAAA//8DANDDSuwNAAAA",
			expected:    []byte("Hello, World!"),
			expectError: false,
		},
		{
			name:        "Plaintext",
			input:       "H4sIAAAAAAAAAwrJSFUoLM1MzlZIKsovz1NIy69QyCrNLShWyC9LLVIoAUrnJFZVKqTkpwMAAAD//wMAOaNPQSsAAAA=",
			expected:    []byte("The quick brown fox jumps over the lazy dog"),
			expectError: false,
		},
		{
			name:        "Base case for compression effectiveness",
			input:       "H4sIAAAAAAAAA0pMJBUAAAAA//8DAH2610cyAAAA",
			expected:    []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			expectError: false,
		},
		{
			name:        "JSON",
			input:       "H4sIAAAAAAAAA6pWyirOz1OyKikqTdVRSs4vzStRsjI0MtZRykstLklNUbKqVkpUslJKUqqtBQAAAP//AwDyGrZhLAAAAA==",
			expected:    []byte("{\"json\":true,\"count\":123,\"nested\":{\"a\":\"b\"}}"),
			expectError: false,
		},
		{
			name:        "Multi-line input",
			input:       "H4sIAAAAAAAAA/LJzEs15PIBkkZg0hhMmgAAAAD//wMAEq75vBcAAAA=",
			expected:    []byte("Line1\nLine2\nLine3\nLine4"),
			expectError: false,
		},
		{
			name:        "De-compressed input",
			input:       "SGVsbG8sIFdvcmxkIQ==",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gzip, err := base64.StdEncoding.DecodeString(tt.input)
			if err != nil {
				t.Fatalf("Test could not complete! (%s)", err.Error())
			}

			res, err := gzipDecode(bytes.NewReader(gzip))

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! (%v)", res)
			}

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestCompressDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "Hello world",
			input:    "Hello, World!",
			expected: []byte("Hello, World!"),
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []byte(""),
		},
		{
			name:     "Single character",
			input:    "A",
			expected: []byte("A"),
		},
		{
			name:     "Repeated pattern (compresses well)",
			input:    "aaaaaabbbbbbcccccc",
			expected: []byte("aaaaaabbbbbbcccccc"),
		},
		{
			name:     "Special characters and numbers",
			input:    "Test123!@# $%^&*()",
			expected: []byte("Test123!@# $%^&*()"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := lzw.NewWriter(&buf, lzw.MSB, 8)
			w.Write([]byte(tt.input))
			w.Close()

			res, err := compressDecode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Errorf("got unexpected error: %s", err.Error())
				return
			}

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestRequestBodyParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		headers     ParsedRequestHeaders
		body        requestBodyParser
		expected    []byte
		expectError bool
	}{
		{
			name: "Hello world",
			headers: ParsedRequestHeaders{
				ContentEncoding: "",
				ContentLength:   13,
			},
			body:        requestBodyParser([]byte("Hello, world!")),
			expected:    []byte("Hello, world!"),
			expectError: false,
		},
		{
			name: "Empty body",
			headers: ParsedRequestHeaders{
				ContentEncoding: "",
				ContentLength:   0,
			},
			body:        requestBodyParser([]byte("")),
			expected:    []byte(""),
			expectError: false,
		},
		{
			name: "Content-Length exceeds body length",
			headers: ParsedRequestHeaders{
				ContentEncoding: "",
				ContentLength:   10,
			},
			body:        requestBodyParser([]byte("abc")),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.body.Parse(tt.headers)

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error! (%v)", res)
			}

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}
