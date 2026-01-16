package message

import (
	"net"
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestRequestParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Minimum valid request",
			data:        []byte("GET / HTTP/1.0\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "No body",
			data:        []byte("GET /index.html HTTP/1.0\r\nHost: example.com\r\nUser-Agent: test\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Single header with folding",
			data:        []byte("GET / HTTP/1.0\r\nUser-Agent: Test\r\n continuation\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Multiple folded headers",
			data:        []byte("GET / HTTP/1.0\r\nX-Test: a\r\n\tb\r\nX-Next: c\r\n d\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Entity body with Content-Length",
			data:        []byte("POST /submit HTTP/1.0\r\nContent-Length: 5\r\n\r\nhello"),
			expectError: false,
		},
		{
			name:        "Headers with strange but legal LWS",
			data:        []byte("GET / HTTP/1.0\r\nX-Test:\tvalue\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "Empty generic header value",
			data:        []byte("GET / HTTP/1.0\r\nX-Empty:\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "No header terminator",
			data:        []byte("GET / HTTP/1.0\r\nHost: example.com\r\n"),
			expectError: true,
		},
		{
			name:        "Missing CRLF after Request-Line",
			data:        []byte("GET / HTTP/1.0\nHost: example.com\r\n\r\n"),
			expectError: true,
		},
		{
			name:        "No header field-name",
			data:        []byte("GET / 1.0\r\n continuation\r\n"),
			expectError: true,
		},
		{
			name:        "Content-Length larger than remaining bytes",
			data:        []byte("POST /submit HTTP/1.0\r\nContent-Length: 10\r\nX-Foo:\t\"Test\"\r\n\r\nhello"),
			expectError: true,
		},
		{
			name:        "?",
			data:        []byte("GET / HTTP/1.0\r\n\r\nHost: example.com\r\n\r\n"),
			expectError: false,
		},
		{
			name:        "??",
			data:        []byte("GET / HTTP/1.0\r\n\r\nhello"),
			expectError: false,
		},
		{
			name:        "Garbage before Request-Line",
			data:        []byte("\r\nGET / HTTP/1.0\r\n\r\n"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			go func() {
				server.Write(tt.data)
			}()

			parser := RequestParser{Connection: client}
			_, err := parser.Parse()
			assert.ErrorStatus(t, err, tt.expectError)
		})
	}
}
