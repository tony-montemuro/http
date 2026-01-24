package http

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"io"
	"testing"
	"time"

	"github.com/tony-montemuro/http/internal/assert"
)

type marshalTest struct {
	name      string
	marshaler marshaler
	expected  []byte
}

func TestResponse_marshal(t *testing.T) {
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
					server: server{
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
					contentType: ContentType{
						Type:    "text",
						Subtype: "plain",
					},
					contentLength: 5,
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
					wwwAuthenticate: challenge{
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
					date: MessageTime{date: t1},
					server: server{
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
					contentType: ContentType{
						Type:    "application",
						Subtype: "octet-stream",
					},
					contentLength: 4,
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
					unrecognized: map[string]string{
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
					date: MessageTime{date: t1},
					server: server{
						products: []ProductVersion{
							{Product: "myserver", Version: "2.1"},
						},
					},
					contentType: ContentType{
						Type:    "text",
						Subtype: "html",
						Parameters: map[string]string{
							"charset": `"utf-8"`,
						},
					},
					contentLength: 13,
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
			res := tt.response.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestCode_marshal(t *testing.T) {
	tests := []struct {
		name        string
		input       code
		expected    []byte
		expectError bool
	}{
		{
			name:     "200 Created",
			input:    StatusOK,
			expected: []byte("HTTP/1.0 200 OK\r\n"),
		},
		{
			name:     "201 Created",
			input:    StatusCreated,
			expected: []byte("HTTP/1.0 201 Created\r\n"),
		},
		{
			name:     "404 Not Found",
			input:    StatusNotFound,
			expected: []byte("HTTP/1.0 404 Not Found\r\n"),
		},
		{
			name:     "301 Moved Permanently",
			input:    StatusMovedPermanently,
			expected: []byte("HTTP/1.0 301 Moved Permanently\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestResponseHeaders_marshal(t *testing.T) {
	t1 := time.Date(2024, 1, 2, 15, 4, 5, 0, time.FixedZone("GMT", 0))
	t2 := time.Date(2023, 12, 25, 0, 0, 0, 0, time.FixedZone("GMT", 0))

	tests := []struct {
		name     string
		headers  responseHeaders
		hasBody  bool
		expected []byte
	}{
		{
			name:     "Empty headers",
			headers:  responseHeaders{},
			expected: []byte("\r\n"),
		},
		{
			name: "Date only",
			headers: responseHeaders{
				date: MessageTime{date: t1},
			},
			expected: []byte(
				"Date: Tue, 02 Jan 2024 15:04:05 GMT\r\n" +
					"\r\n",
			),
		},
		{
			name: "Server only",
			headers: responseHeaders{
				server: server{
					products: []ProductVersion{
						{Product: "myserver", Version: "1.0"},
					},
				},
			},
			expected: []byte(
				"Server: myserver/1.0\r\n" +
					"\r\n",
			),
		},
		{
			name: "Pragma and Content-Length",
			headers: responseHeaders{
				pragma: PragmaDirectives{
					Flags: map[string]bool{"no-cache": true},
				},
				contentLength: 123,
			},
			hasBody: true,
			expected: []byte(
				"Pragma: no-cache\r\n" +
					"Content-Length: 123\r\n" +
					"\r\n",
			),
		},
		{
			name: "WWW-Authenticate only with body",
			headers: responseHeaders{
				wwwAuthenticate: challenge{
					scheme: "Basic",
					realm:  `"Restricted"`,
				},
			},
			hasBody: true,
			expected: []byte(
				`WWW-Authenticate: Basic realm="Restricted"` + "\r\n" +
					"Content-Length: 0\r\n" +
					"\r\n",
			),
		},
		{
			name: "Allow header",
			headers: responseHeaders{
				allow: Methods{
					methods: []Method{"GET", "HEAD", "POST"},
				},
			},
			expected: []byte(
				"Allow: GET, HEAD, POST\r\n" +
					"\r\n",
			),
		},
		{
			name: "Content-Type with parameters and body",
			headers: responseHeaders{
				contentType: ContentType{
					Type:    "text",
					Subtype: "html",
					Parameters: map[string]string{
						"charset": `"utf-8"`,
					},
				},
			},
			hasBody: true,
			expected: []byte(
				"Content-Length: 0\r\n" +
					`Content-Type: text/html;charset="utf-8"` +
					"\r\n" + "\r\n",
			),
		},
		{
			name: "Content-Encoding",
			headers: responseHeaders{
				contentEncoding: "x-gzip",
			},
			expected: []byte(
				"Content-Encoding: x-gzip\r\n" +
					"\r\n",
			),
		},
		{
			name: "Expires and Last-Modified",
			headers: responseHeaders{
				expires:      MessageTime{date: t2},
				lastModified: MessageTime{date: t1},
			},
			expected: []byte(
				"Expires: Mon, 25 Dec 2023 00:00:00 GMT\r\n" +
					"Last-Modified: Tue, 02 Jan 2024 15:04:05 GMT\r\n" +
					"\r\n",
			),
		},
		{
			name: "Unrecognized headers mixed with known",
			headers: responseHeaders{
				contentLength: 42,
				unrecognized: map[string]string{
					"X-Foo": "bar",
					"X-Baz": "qux",
				},
			},
			hasBody: true,
			expected: []byte(
				"Content-Length: 42\r\n" +
					"X-Baz: qux\r\n" +
					"X-Foo: bar\r\n" +
					"\r\n",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.headers.marshal(tt.hasBody)
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestMessageTime_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name:      "Basic example",
			marshaler: MessageTime{date: time.Unix(0, 0).In(time.FixedZone("GMT", 0))},
			expected:  []byte("Thu, 01 Jan 1970 00:00:00 GMT"),
		},
		{
			name:      "RFC example",
			marshaler: MessageTime{date: time.Date(1994, time.November, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0))},
			expected:  []byte("Sun, 06 Nov 1994 08:49:37 GMT"),
		},
		{
			name:      "Leap day",
			marshaler: MessageTime{date: time.Date(2020, time.February, 29, 12, 34, 56, 0, time.FixedZone("GMT", 0))},
			expected:  []byte("Sat, 29 Feb 2020 12:34:56 GMT"),
		},
		{
			name:      "Future date",
			marshaler: MessageTime{date: time.Date(2099, time.December, 31, 23, 59, 59, 0, time.FixedZone("GMT", 0))},
			expected:  []byte("Thu, 31 Dec 2099 23:59:59 GMT"),
		},
		{
			name:      "Zero date",
			marshaler: MessageTime{date: time.Time{}},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestPragmaDirectives_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "No cache",
			marshaler: PragmaDirectives{
				Flags: map[string]bool{"no-cache": true},
			},
			expected: []byte("no-cache"),
		},
		{
			name: "Multiple flags",
			marshaler: PragmaDirectives{
				Flags: map[string]bool{
					"no-cache": true,
					"foo":      true,
					"bar":      true,
				},
			},
			expected: []byte("bar foo no-cache"),
		},
		{
			name: "Options only",
			marshaler: PragmaDirectives{
				Options: map[string]string{
					"ttl":  "60",
					"mode": "fast",
				},
			},
			expected: []byte("mode=fast ttl=60"),
		},
		{
			name: "Flags & options",
			marshaler: PragmaDirectives{
				Flags: map[string]bool{"no-cache": true},
				Options: map[string]string{
					"ttl": "30",
				},
			},
			expected: []byte("no-cache ttl=30"),
		},
		{
			name:      "Empty directives",
			marshaler: PragmaDirectives{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestProductVersion_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Single product with version",
			marshaler: ProductVersion{
				Product: "CERN",
				Version: "3.0",
			},
			expected: []byte("CERN/3.0"),
		},
		{
			name: "Product without version",
			marshaler: ProductVersion{
				Product: "go",
			},
			expected: []byte("go"),
		},
		{
			name:      "No product or version",
			marshaler: ProductVersion{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestAbsoluteUri_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Standard HTTP",
			marshaler: &AbsoluteUri{
				Scheme: []byte("http"),
				Path:   []byte("//example.com/index.html"),
			},
			expected: []byte("http://example.com/index.html"),
		},
		{
			name: "Secure HTTPS",
			marshaler: &AbsoluteUri{
				Scheme: []byte("https"),
				Path:   []byte("//secure.site/login"),
			},
			expected: []byte("https://secure.site/login"),
		},
		{
			name: "Mailto link",
			marshaler: &AbsoluteUri{
				Scheme: []byte("mailto"),
				Path:   []byte("user@domain.com"),
			},
			expected: []byte("mailto:user@domain.com"),
		},
		{
			name: "Empty Path",
			marshaler: &AbsoluteUri{
				Scheme: []byte("news"),
				Path:   []byte(""),
			},
			expected: []byte("news:"),
		},
		{
			name: "Complex Scheme",
			marshaler: &AbsoluteUri{
				Scheme: []byte("soap-beep+v2"),
				Path:   []byte("//api/endpoint"),
			},
			expected: []byte("soap-beep+v2://api/endpoint"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestRelativeUri_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Simple Path",
			marshaler: &RelativeUri{
				Path: []byte("images/logo.png"),
			},
			expected: []byte("images/logo.png"),
		},
		{
			name: "Rooted Path",
			marshaler: &RelativeUri{
				Path: []byte("/usr/local/bin"),
			},
			expected: []byte("/usr/local/bin"),
		},
		{
			name: "Network Location with Path",
			marshaler: &RelativeUri{
				NetLoc: []byte("example.com"),
				Path:   []byte("/home"),
			},
			expected: []byte("//example.com/home"),
		},
		{
			name: "Path with Parameters",
			marshaler: &RelativeUri{
				Path: []byte("item"),
				Params: [][]byte{
					[]byte("version=1"),
					[]byte("format=json"),
				},
			},
			expected: []byte("item;version=1;format=json"),
		},
		{
			name: "Path with Query",
			marshaler: &RelativeUri{
				Path:  []byte("/search"),
				Query: []byte("q=golang"),
			},
			expected: []byte("/search?q=golang"),
		},
		{
			name: "Query Only",
			marshaler: &RelativeUri{
				Query: []byte("page=5"),
			},
			expected: []byte("?page=5"),
		},
		{
			name: "All Components",
			marshaler: &RelativeUri{
				NetLoc: []byte("api.srv"),
				Path:   []byte("/v1/user"),
				Params: [][]byte{
					[]byte("auth=token"),
				},
				Query: []byte("debug=true"),
			},
			expected: []byte("//api.srv/v1/user;auth=token?debug=true"),
		},
		{
			name: "NetLoc Only",
			marshaler: &RelativeUri{
				NetLoc: []byte("localhost:8080"),
			},
			expected: []byte("//localhost:8080"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestServer_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Multiple products with versions",
			marshaler: server{
				products: []ProductVersion{
					{Product: "CERN", Version: "3.0"},
					{Product: "libwww", Version: "2.17"},
				},
			},
			expected: []byte("CERN/3.0 libwww/2.17"),
		},
		{
			name: "Products and comments",
			marshaler: server{
				products: []ProductVersion{
					{Product: "MyServer", Version: "1.2.3"},
				},
				comments: []string{
					"(Unix)",
					"(Experimental)",
				},
			},
			expected: []byte("MyServer/1.2.3 (Unix) (Experimental)"),
		},
		{
			name:      "Empty server header",
			marshaler: server{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestChallenge_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Basic auth with realm only",
			marshaler: challenge{
				scheme: "Basic",
				realm:  `"Restricted"`,
			},
			expected: []byte(`Basic realm="Restricted"`),
		},
		{
			name: "Basic auth with realm and one param",
			marshaler: challenge{
				scheme: "Basic",
				realm:  `"Restricted"`,
				params: map[string]string{
					"charset": `"UTF-8"`,
				},
			},
			expected: []byte(`Basic realm="Restricted",charset="UTF-8"`),
		},
		{
			name: "Digest auth with multiple params",
			marshaler: challenge{
				scheme: "Digest",
				realm:  `"api"`,
				params: map[string]string{
					"nonce": `"abc123"`,
					"qop":   `"auth"`,
				},
			},
			expected: []byte(`Digest realm="api",nonce="abc123",qop="auth"`),
		},
		{
			name: "Custom scheme with custom params",
			marshaler: challenge{
				scheme: "Token",
				realm:  `"users"`,
				params: map[string]string{
					"issuer": `"auth.example.com"`,
				},
			},
			expected: []byte(`Token realm="users",issuer="auth.example.com"`),
		},
		{
			name: "Realm with spaces",
			marshaler: challenge{
				scheme: "Basic",
				realm:  `"Admin Area"`,
			},
			expected: []byte(`Basic realm="Admin Area"`),
		},
		{
			name:      "Empty challenge",
			marshaler: challenge{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestAllow_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Single method",
			marshaler: Methods{
				methods: []Method{"GET"},
			},
			expected: []byte("GET"),
		},
		{
			name: "Many methods",
			marshaler: Methods{
				methods: []Method{"GET", "POST", "PUT", "DELETE"},
			},
			expected: []byte("GET, POST, PUT, DELETE"),
		},
		{
			name: "Custom/extension method",
			marshaler: Methods{
				methods: []Method{"GET", "FOO"},
			},
			expected: []byte("GET, FOO"),
		},
		{
			name:      "Empty methods",
			marshaler: Methods{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestContentEncoding_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name:      "x-gzip encoding",
			marshaler: ContentEncoding("x-gzip"),
			expected:  []byte("x-gzip"),
		},
		{
			name:      "custom token encoding",
			marshaler: ContentEncoding("br"),
			expected:  []byte("br"),
		},
		{
			name:      "Empty encoding",
			marshaler: ContentEncoding(""),
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestContentLength_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name:      "Zero length",
			marshaler: ContentLength(0),
			expected:  []byte("0"),
		},
		{
			name:      "Small content length",
			marshaler: ContentLength(42),
			expected:  []byte("42"),
		},
		{
			name:      "Typical payload size",
			marshaler: ContentLength(3495),
			expected:  []byte("3495"),
		},
		{
			name:      "Large content length",
			marshaler: ContentLength(123456789),
			expected:  []byte("123456789"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestContentType_marshal(t *testing.T) {
	tests := []marshalTest{
		{
			name: "Simple type and subtype",
			marshaler: ContentType{
				Type:    "text",
				Subtype: "html",
			},
			expected: []byte("text/html"),
		},
		{
			name: "Type with single token parameter",
			marshaler: ContentType{
				Type:    "text",
				Subtype: "plain",
				Parameters: map[string]string{
					"charset": "utf-8",
				},
			},
			expected: []byte("text/plain;charset=utf-8"),
		},
		{
			name: "Type with quoted-string parameter",
			marshaler: ContentType{
				Type:    "text",
				Subtype: "html",
				Parameters: map[string]string{
					"charset": `"iso-8859-1"`,
				},
			},
			expected: []byte(`text/html;charset="iso-8859-1"`),
		},
		{
			name: "Multiple parameters mixed token and quoted-string",
			marshaler: ContentType{
				Type:    "multipart",
				Subtype: "form-data",
				Parameters: map[string]string{
					"boundary": `"----WebKitFormBoundaryABC123"`,
					"charset":  "utf-8",
				},
			},
			expected: []byte(`multipart/form-data;boundary="----WebKitFormBoundaryABC123";charset=utf-8`),
		},
		{
			name:      "Empty Content-Type",
			marshaler: ContentType{},
			expected:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.marshaler.marshal()
			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestEncodeRequestBody(t *testing.T) {
	// gzip1, err := base64.StdEncoding.DecodeString("")
	// if err != nil {
	// 	t.Fatalf("Test could not complete! (%s)", err.Error())
	// }
	//
	// var buf bytes.Buffer
	// w := lzw.NewWriter(&buf, lzw.MSB, 8)
	// _, err = w.Write([]byte("Hello, World!"))
	// if err != nil {
	// 	t.Fatalf("Test could not complete! (%s)", err.Error())
	// }
	//
	// err = w.Close()
	// if err != nil {
	// 	t.Fatalf("Test could not complete! (%s)", err.Error())
	// }

	tests := []struct {
		name        string
		body        []byte
		encoding    ContentEncoding
		expectError bool
	}{
		{
			name:        "Unencoded hello world",
			body:        []byte("Hello, world!"),
			encoding:    ContentEncoding(""),
			expectError: false,
		},
		{
			name:        "Unencoded empty body",
			body:        []byte(""),
			encoding:    ContentEncoding(""),
			expectError: false,
		},
		{
			name:        "Gzip encoded empty body",
			body:        []byte(""),
			encoding:    ContentEncoding("x-gzip"),
			expectError: false,
		},
		{
			name:        "Compress encoded empty body",
			body:        []byte(""),
			encoding:    ContentEncoding("x-compress"),
			expectError: false,
		},
		{
			name:        "Gzip non-empty body",
			body:        []byte("some data"),
			encoding:    ContentEncoding("gzip"),
			expectError: false,
		},
		{
			name:        "Compress non-empty body",
			body:        []byte("some more data"),
			encoding:    ContentEncoding("compress"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded []byte
			res, err := encodeRequestBody(tt.body, tt.encoding)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}
			reader := bytes.NewReader(res)

			switch tt.encoding {
			case ContentEncodingGZip, ContentEncodingXGzip:
				reader, err := gzip.NewReader(reader)
				if err != nil {
					t.Fatalf("Test could not complete! (%s)", err.Error())
				}
				defer reader.Close()

				decoded, err = io.ReadAll(reader)
				if err != nil {
					t.Fatalf("Test could not complete! (%s)", err.Error())
				}
			case ContentEncodingCompress, ContentEncodingXCompress:
				reader := lzw.NewReader(reader, lzw.LSB, 8)
				defer reader.Close()

				decoded, err = io.ReadAll(reader)
				if err != nil {
					t.Fatalf("Test could not complete! (%s)", err.Error())
				}
			default:
				decoded = res
			}

			assert.SliceEqual(t, decoded, tt.body)
		})
	}
}
