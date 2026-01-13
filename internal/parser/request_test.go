package parser

import (
	"testing"
)

func TestRequestParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		request     []byte
		expectError bool
	}{
		{
			name:        "Minimum valid request",
			request:     []byte("GET / HTTP/1.0\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "No body",
			request:     []byte("GET /index.html HTTP/1.0\r\nHost: example.com\r\nUser-Agent: test\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Single header with folding",
			request:     []byte("GET / HTTP/1.0\r\nUser-Agent: Test\r\n continuation\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Multiple folded headers",
			request:     []byte("GET / HTTP/1.0\r\nX-Test: a\r\n\tb\r\nX-Next: c\r\n d\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Entity body with Content-Length",
			request:     []byte("POST /submit HTTP/1.0\r\nContent-Length: 5\r\n\r\nhello"),
			expectError: false,
		},
		{
			name:        "Headers with strange but legal LWS",
			request:     []byte("GET / HTTP/1.0\r\nX-Test:\tvalue\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Empty generic header value",
			request:     []byte("GET / HTTP/1.0\r\nX-Empty:\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "No header terminator",
			request:     []byte("GET / HTTP/1.0\r\nHost: example.com\r\n"),
			expectError: true,
		},
		{
			name:        "Missing CRLF after Request-Line",
			request:     []byte("GET / HTTP/1.0\nHost: example.com\r\n\r\n"),
			expectError: true,
		},
		{
			name:        "No header field-name",
			request:     []byte("GET / 1.0\r\n continuation\r\n"),
			expectError: true,
		},
		{
			name:        "Content-Length larger than remaining bytes",
			request:     []byte("POST /submit HTTP/1.0\r\nContent-Length: 10\r\nX-Foo:\t\"Test\"\r\n\r\nhello"),
			expectError: true,
		},
		{
			name:        "?",
			request:     []byte("GET / HTTP/1.0\r\n\r\nHost: example.com\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "??",
			request:     []byte("GET / HTTP/1.0\r\n\r\nhello"),
			expectError: false,
		},
		{
			name:        "Garbage before Request-Line",
			request:     []byte("\r\nGET / HTTP/1.0\r\n\r\n"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RequestParser(tt.request).Parse()

			if err != nil {
				if !tt.expectError {
					t.Errorf("got unexpected error: %s", err.Error())
				}
				return
			}

			if tt.expectError {
				t.Errorf("did not get expected error!")
			}
		})
	}
}
