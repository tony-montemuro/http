package message

import (
	"testing"
	"time"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestResponse_Marshal(t *testing.T) {
	t1 := time.Date(2024, 1, 2, 15, 4, 5, 0, time.FixedZone("GMT", 0))

	tests := []struct {
		name     string
		response response
		expected []byte
	}{
		{
			name: "Minimal response",
			response: response{
				code: 200,
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"\r\n",
			),
		},
		{
			name: "Response with body only",
			response: response{
				code: 200,
				body: responseBody("hello world"),
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"Content-Length: 0\r\n" +
					"\r\n" +
					"hello world",
			),
		},
		{
			name: "Response with headers only",
			response: response{
				code: 204,
				headers: responseHeaders{
					Server: server{
						products: []ProductVersion{
							{Product: "go"},
						},
					},
				},
			},
			expected: []byte(
				"HTTP/1.0 204 No Content\r\n" +
					"Server: go\r\n" +
					"\r\n",
			),
		},
		{
			name: "Response with headers and body",
			response: response{
				code: 200,
				headers: responseHeaders{
					ContentType: ContentType{
						Type:    "text",
						Subtype: "plain",
					},
					ContentLength: 5,
				},
				body: responseBody("hello"),
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"Content-Length: 5\r\n" +
					"Content-Type: text/plain\r\n" +
					"\r\n" +
					"hello",
			),
		},
		{
			name: "401 with WWW-Authenticate",
			response: response{
				code: 401,
				headers: responseHeaders{
					WwwAuthenticate: challenge{
						scheme: "Basic",
						realm:  `"Restricted"`,
					},
				},
			},
			expected: []byte(
				"HTTP/1.0 401 Unauthorized\r\n" +
					`WWW-Authenticate: Basic realm="Restricted"` + "\r\n" +
					"\r\n",
			),
		},
		{
			name: "Server and Date headers",
			response: response{
				code: 200,
				headers: responseHeaders{
					Date: MessageTime{date: t1},
					Server: server{
						products: []ProductVersion{
							{Product: "example", Version: "1.0"},
						},
					},
				},
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"Date: Tue, 02 Jan 2024 15:04:05 GMT\r\n" +
					"Server: example/1.0\r\n" +
					"\r\n",
			),
		},
		{
			name: "Binary body",
			response: response{
				code: 200,
				headers: responseHeaders{
					ContentType: ContentType{
						Type:    "application",
						Subtype: "octet-stream",
					},
					ContentLength: 4,
				},
				body: responseBody([]byte{0x00, 0x01, 0x02, 0xFF}),
			},
			expected: append(
				[]byte(
					"HTTP/1.0 200 OK\r\n"+
						"Content-Length: 4\r\n"+
						"Content-Type: application/octet-stream\r\n"+
						"\r\n",
				),
				[]byte{0x00, 0x01, 0x02, 0xFF}...,
			),
		},
		{
			name: "Unrecognized headers included",
			response: response{
				code: 200,
				headers: responseHeaders{
					Unrecognized: map[string]string{
						"X-Test": "abc",
					},
				},
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"X-Test: abc\r\n" +
					"\r\n",
			),
		},
		{
			name: "204 No Content with body largely ignored",
			response: response{
				code: 204,
				body: responseBody("should not matter"),
			},
			expected: []byte(
				"HTTP/1.0 204 No Content\r\n" +
					"Content-Length: 0\r\n" +
					"\r\n" +
					"should not matter",
			),
		},
		{
			name: "Complex realistic response",
			response: response{
				code: 200,
				headers: responseHeaders{
					Date: MessageTime{date: t1},
					Server: server{
						products: []ProductVersion{
							{Product: "myserver", Version: "2.1"},
						},
					},
					ContentType: ContentType{
						Type:    "text",
						Subtype: "html",
						Parameters: map[string]string{
							"charset": `"utf-8"`,
						},
					},
					ContentLength: 13,
				},
				body: responseBody("<h1>Hello</h1>"),
			},
			expected: []byte(
				"HTTP/1.0 200 OK\r\n" +
					"Date: Tue, 02 Jan 2024 15:04:05 GMT\r\n" +
					"Server: myserver/2.1\r\n" +
					"Content-Length: 13\r\n" +
					`Content-Type: text/html;charset="utf-8"` + "\r\n" +
					"\r\n" +
					"<h1>Hello</h1>",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.response.Marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}
