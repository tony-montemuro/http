package message

import (
	"net/mail"
	"testing"
	"time"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestRequestHeaderParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:  "Minimal valid headers",
			input: "Allow: GET",
			expected: RequestHeaders{
				Allow: []Method{"GET"},
				raw: map[string]string{
					"Allow": "GET",
				},
			},
			expectError: false,
		},
		{
			name:  "Multiple standard headers",
			input: "Allow: GET\r\nUser-Agent: Client/1.0\r\nDate: Sun, 06 Nov 1994 08:49:37 GMT",
			expected: RequestHeaders{
				Allow: []Method{"GET"},
				UserAgent: UserAgent{
					Products: []ProductVersion{
						{Product: "Client", Version: "1.0"},
					},
				},
				Date: time.Date(1994, 11, 6, 8, 49, 37, 0, time.FixedZone("GMT", 0)),
				raw: map[string]string{
					"Allow":      "GET",
					"User-Agent": "Client/1.0",
					"Date":       "Sun, 06 Nov 1994 08:49:37 GMT",
				},
			},
			expectError: false,
		},
		{
			name:  "Unknown header",
			input: "X-Weird-Header: some-value",
			expected: RequestHeaders{
				Unrecognized: map[string]string{
					"X-Weird-Header": "some-value",
				},
				raw: map[string]string{
					"X-Weird-Header": "some-value",
				},
			},
			expectError: false,
		},
		{
			name:  "Header with empty value",
			input: "X-Foo:",
			expected: RequestHeaders{
				Unrecognized: map[string]string{
					"X-Foo": "",
				},
				raw: map[string]string{
					"X-Foo": "",
				},
			},
		},
		{
			name:  "Excessive but valid LWS",
			input: "Content-Type  \r\n  :\t\t text/html   ;\r\n charset=UTF-8\r\n\t",
			expected: RequestHeaders{
				ContentType: ContentType{
					Type:    "text",
					Subtype: "html",
					Parameters: map[string]string{
						"charset": "UTF-8",
					},
				},
				raw: map[string]string{
					"Content-Type": "text/html   ;\r\n charset=UTF-8\r\n\t",
				},
			},
			expectError: false,
		},
		{
			name:        "Bad header",
			input:       "Bad Header: value",
			expectError: true,
		},
		{
			name:        "Control character in field-value",
			input:       "X-Test: hello\x01world",
			expectError: true,
		},
		{
			name:  "Duplicate header fields",
			input: "Content-Length:\t35\r\nContent-Length: 36",
			expected: RequestHeaders{
				ContentLength: uint64(36),
				raw: map[string]string{
					"Content-Length": "36",
				},
			},
			expectError: false,
		},
		{
			name:        "Mixed valid + invalid headers",
			input:       "Host: example.com\r\nContent-Length: 10\r\nBad Header@: reject",
			expectError: true,
		},
		{
			name:        "No headers",
			input:       "",
			expected:    RequestHeaders{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := requestHeadersParser(tt.input).Parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.DateEqual(t, res.Date, tt.expected.Date)
			assert.SliceEqual(t, res.Pragma.Flags, tt.expected.Pragma.Flags)
			assert.MapEqual(t, res.Pragma.Options, tt.expected.Pragma.Options)
			assert.Equal(t, res.Authorization.Scheme, tt.expected.Authorization.Scheme)
			assert.MapEqual(t, res.Authorization.Parameters, tt.expected.Authorization.Parameters)
			assert.Equal(t, res.From.Name, tt.expected.From.Name)
			assert.Equal(t, res.From.Address, tt.expected.From.Address)
			assert.DateEqual(t, res.IfModifiedSince, tt.expected.IfModifiedSince)
			assert.Equal(t, res.Referer, tt.expected.Referer)

			assert.SliceEqual(t, res.UserAgent.Comments, tt.expected.UserAgent.Comments)
			expectedProducts := tt.expected.UserAgent.Products
			actualProducts := res.UserAgent.Products

			if len(expectedProducts) != len(actualProducts) {
				t.Errorf("different sizes. got: (%v, len: %d), want: (%v, len: %d)", actualProducts, len(actualProducts), expectedProducts, len(expectedProducts))
				return
			}

			for i := range len(actualProducts) {
				assert.Equal(t, actualProducts[i].Product, expectedProducts[i].Product)
				assert.Equal(t, actualProducts[i].Version, expectedProducts[i].Version)
			}

			assert.SliceEqual(t, res.Allow, tt.expected.Allow)
			assert.Equal(t, res.ContentEncoding, tt.expected.ContentEncoding)
			assert.Equal(t, res.ContentLength, tt.expected.ContentLength)
			assert.Equal(t, res.ContentType.Type, tt.expected.ContentType.Type)
			assert.Equal(t, res.ContentType.Subtype, tt.expected.ContentType.Subtype)
			assert.MapEqual(t, res.ContentType.Parameters, tt.expected.ContentType.Parameters)
			assert.DateEqual(t, res.Expires, tt.expected.Expires)
			assert.DateEqual(t, res.LastModified, tt.expected.LastModified)
			assert.MapEqual(t, res.Unrecognized, tt.expected.Unrecognized)
			assert.MapEqual(t, res.raw, tt.expected.raw)
		})
	}
}

func TestHeaderSplitter_split(t *testing.T) {
	tests := []struct {
		name     string
		headers  []byte
		expected [][]byte
	}{
		{
			name:     "Single header",
			headers:  []byte("example.com"),
			expected: [][]byte{[]byte("example.com")},
		},
		{
			name:     "Multiple headers",
			headers:  []byte("Host: example.com\r\nUser-Agent: test\r\nAccept: */*"),
			expected: [][]byte{[]byte("Host: example.com"), []byte("User-Agent: test"), []byte("Accept: */*")},
		},
		{
			name:     "Header with folded continuation",
			headers:  []byte("abc\r\n def"),
			expected: [][]byte{[]byte("abc\r\n def")},
		},
		{
			name:     "Header with multiple folded lines",
			headers:  []byte("abc\r\n def\r\n\tghi"),
			expected: [][]byte{[]byte("abc\r\n def\r\n\tghi")},
		},
		{
			name:     "Folded header followed by new header",
			headers:  []byte("abc\r\n def\r\nOther: value"),
			expected: [][]byte{[]byte("abc\r\n def"), []byte("Other: value")},
		},
		{
			name:     "Empty header",
			headers:  []byte(""),
			expected: [][]byte{},
		},
		{
			name:     "Header with internal LWS without folding",
			headers:  []byte("abc def ghi"),
			expected: [][]byte{[]byte("abc def ghi")},
		},
		{
			name:     "Multiple headers, mixed folding",
			headers:  []byte("A: one\r\nB: two\r\n three\r\nC: four"),
			expected: [][]byte{[]byte("A: one"), []byte("B: two\r\n three"), []byte("C: four")},
		},
	}

	for _, tt := range tests {
		assert.MatrixEqual(t, headerSplitter(tt.headers).split(), tt.expected)
	}
}

func TestHeaderNameValidator_validate(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Standard header name (Date)",
			token:       "Date",
			expectError: false,
		},
		{
			name:        "String with control characters (def\n456)",
			token:       "def\n456",
			expectError: true,
		},
		{
			name:        "String with extended ASCII characters (ghi[200]789)",
			token:       string([]byte{'g', 'h', 'i', 200, '7', '8', '9'}),
			expectError: true,
		},
		{
			name:        "String with TSpecial characters (jkl\\098)",
			token:       "jkl\\098",
			expectError: true,
		},
		{
			name:        "Empty string",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := headerNameValidator(tt.token).validate()
			assert.ErrorStatus(t, err, tt.expectError)
		})
	}
}

func TestHeaderValueValidator_validate(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{
			name:        "Simple text",
			value:       "Hello world",
			expectError: false,
		},
		{
			name:        "Leading and trailing spaces",
			value:       "   value   ",
			expectError: false,
		},
		{
			name:        "Folded line",
			value:       "abc\r\n def",
			expectError: false,
		},
		{
			name:        "Multiple LWS folds",
			value:       "abc\r\n\t def\r\n \t\tghi",
			expectError: false,
		},
		{
			name:        "Quoted string",
			value:       "\"this is good\"",
			expectError: false,
		},
		{
			name:        "tspecials present",
			value:       "foo; bar, baz",
			expectError: false,
		},
		{
			name:        "Control character",
			value:       "abc\x00def",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := headerValueValidator(tt.value).validate()
			assert.ErrorStatus(t, err, tt.expectError)
		})
	}
}

func TestPragmaHeaderParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		pragmaVal   string
		expected    PragmaDirectives
		expectError bool
	}{
		{
			name:        "Standard case (no-cache)",
			pragmaVal:   "no-cache",
			expected:    PragmaDirectives{Flags: []string{"no-cache"}},
			expectError: false,
		},
		{
			name:        "Single extension (foo)",
			pragmaVal:   "foo",
			expected:    PragmaDirectives{Flags: []string{"foo"}},
			expectError: false,
		},
		{
			name:        "Single extension with value (foo=bar)",
			pragmaVal:   "foo=bar",
			expected:    PragmaDirectives{Options: map[string]string{"foo": "bar"}},
			expectError: false,
		},
		{
			name:        "Multiple directives (no-cache, foo=bar)",
			pragmaVal:   "no-cache, foo=bar",
			expected:    PragmaDirectives{Flags: []string{"no-cache"}, Options: map[string]string{"foo": "bar"}},
			expectError: false,
		},
		{
			name:        "Extra whitespace (  no-cache \t,  \t foo=bar ,     flag)",
			pragmaVal:   "  no-cache \t,  \t foo=bar ,     flag",
			expected:    PragmaDirectives{Flags: []string{"no-cache", "flag"}, Options: map[string]string{"foo": "bar"}},
			expectError: false,
		},
		{
			name:        "Multiple flags and options (no-cache, foo=bar, baz, this=works)",
			pragmaVal:   "no-cache, foo=bar, baz, this=works",
			expected:    PragmaDirectives{Flags: []string{"no-cache", "baz"}, Options: map[string]string{"foo": "bar", "this": "works"}},
			expectError: false,
		},
		{
			name:        "No-cache with value (no-cache=1)",
			pragmaVal:   "no-cache=1",
			expectError: true,
		},
		{
			name:        "Whitespace around equals (foo = bar)",
			pragmaVal:   "foo = bar",
			expectError: true,
		},
		{
			name:        "Trailing comma (no-cache,)",
			pragmaVal:   "no-cache,",
			expectError: true,
		},
		{
			name:        "Empty value ()",
			pragmaVal:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := pragmaHeaderParser(tt.pragmaVal).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, res.Flags, tt.expected.Flags)
			assert.MapEqual(t, res.Options, tt.expected.Options)
		})
	}
}

func TestAuthorizationHeaderSplitter_split(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "Standard case",
			value:    "Digest realm=\"example\"",
			expected: []string{"Digest", "realm=\"example\""},
		},
		{
			name:     "Non-standard delimeter",
			value:    "Digest?realm=\"example\"",
			expected: []string{"Digest", "realm=\"example\""},
		},
		{
			name:     "Non-tspecial LWS preceeding a delimeter",
			value:    "Digest\r\n  realm=\"example\"",
			expected: []string{"Digest\r\n ", "realm=\"example\""},
		},
		{
			name:     "Multiple non-tspecail LWS preceeding a delimeter",
			value:    "Digest\r\n \r\n\t\t realm=\"example\"",
			expected: []string{"Digest\r\n \r\n\t", " realm=\"example\""},
		},
		{
			name:     "Sequence containing multiple LWS but no delimieters",
			value:    "Dig\r\n est\r\n\trealm_example",
			expected: []string{"Dig\r\n est\r\n\trealm_example", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.SliceEqual(t, authorizationHeaderSplitter(tt.value).split(), tt.expected)
		})
	}
}

func TestAuthorizationHeaderParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expected    AuthorizationCredentials
		expectError bool
	}{
		{
			name:  "Common header",
			value: "Digest realm=\"example\"",
			expected: AuthorizationCredentials{
				Scheme:     "Digest",
				Parameters: map[string]string{"realm": "example"},
			},
			expectError: false,
		},
		{
			name:  "Multiple params, common form",
			value: "Digest realm=\"a\", nonce=\"b\"",
			expected: AuthorizationCredentials{
				Scheme: "Digest",
				Parameters: map[string]string{
					"realm": "a",
					"nonce": "b",
				},
			},
			expectError: false,
		},
		{
			name:  "Extra LWS before params",
			value: "Digest  \r\n\trealm=\"example\"",
			expected: AuthorizationCredentials{
				Scheme:     "Digest",
				Parameters: map[string]string{"realm": "example"},
			},
		},
		{
			name:  "Extra LWS separating multiple parameters",
			value: "Digest\r\n (\r\n\t\t \r\n realm=\"a\" ,\t\r\n\tnonce=\"b\"",
			expected: AuthorizationCredentials{
				Scheme: "Digest",
				Parameters: map[string]string{
					"realm": "a",
					"nonce": "b",
				},
			},
		},
		{
			name:  "Basic authorization",
			value: "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "Aladdin",
					"password": "open sesame",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := authorizationHeaderParser(tt.value).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res.Scheme, tt.expected.Scheme)
			assert.MapEqual(t, res.Parameters, tt.expected.Parameters)
		})
	}
}

func TestAuthorizationCredentials_setBasicSchemeParams(t *testing.T) {
	tests := []struct {
		name        string
		cookie      string
		expected    AuthorizationCredentials
		expectError bool
	}{
		{
			name:   "Empty userid",
			cookie: "OnBhc3N3b3Jk",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "",
					"password": "password",
				},
			},
		},
		{
			name:   "Empty password",
			cookie: "QWxhZGRpbjo=",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "Aladdin",
					"password": "",
				},
			},
			expectError: false,
		},
		{
			name:   "Empty userid and password",
			cookie: "Og==",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "",
					"password": "",
				},
			},
			expectError: false,
		},
		{
			name:   "More advanced userid",
			cookie: "dXNlci0xMjM0OnBhc3M=",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "user-1234",
					"password": "pass",
				},
			},
			expectError: false,
		},
		{
			name:        "Invalid userid",
			cookie:      "dXNlciBuYW1lOnBhc3M=",
			expectError: true,
		},
		{
			name:        "Missing colon",
			cookie:      "QWxhZGRpbl9vcGVuIHNlc2FtZQ==",
			expectError: true,
		},
		{
			name:   "Multiple colons",
			cookie: "dXNlcjpwYXNzOndpdGg6Y29sb25z",
			expected: AuthorizationCredentials{
				Scheme: "Basic",
				Parameters: map[string]string{
					"userid":   "user",
					"password": "pass:with:colons",
				},
			},
			expectError: false,
		},
		{
			name:        "Invalid base64 input",
			cookie:      "!!!not-base64!!!",
			expectError: true,
		},
		{
			name:        "Password contains control character",
			cookie:      "dXNlcjpwYXNzAA==",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := AuthorizationCredentials{Scheme: "Basic"}
			err := credentials.setParams(tt.cookie)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, credentials.Scheme, tt.expected.Scheme)
			assert.MapEqual(t, credentials.Parameters, tt.expected.Parameters)
		})
	}
}

func TestRequestHeaders_setFrom(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:  "Simple email address",
			email: "user@example.com",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "",
					Address: "user@example.com",
				},
			},
			expectError: false,
		},
		{
			name:  "Full email address",
			email: "User <user@example.com>",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "User",
					Address: "user@example.com",
				},
			},
			expectError: false,
		},
		{
			name:  "Quoted display name",
			email: "\"User Name\" <user@example.com>",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "User Name",
					Address: "user@example.com",
				},
			},
			expectError: false,
		},
		{
			name:  "Quoted display name with comma",
			email: "\"Last, First\" <user@example.com>",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "Last, First",
					Address: "user@example.com",
				},
			},
			expectError: false,
		},
		{
			name:  "Address with subdomain",
			email: "user@mail.example.com",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "",
					Address: "user@mail.example.com",
				},
			},
			expectError: false,
		},
		{
			name:  "Extra preceding whitespace",
			email: " \t   User <user@example.com>",
			expected: RequestHeaders{
				From: mail.Address{
					Name:    "User",
					Address: "user@example.com",
				},
			},
			expectError: false,
		},
		{
			name:        "Empty value",
			email:       "",
			expectError: true,
		},
		{
			name:        "Missing angle brackets around address",
			email:       "User user@example.com",
			expectError: true,
		},
		{
			name:        "Invalid email (no domain)",
			email:       "User <user@>",
			expectError: true,
		},
		{
			name:        "Multiple addresses (prohibited)",
			email:       "user@example.com, user2@example.com",
			expectError: true,
		},
		{
			name:        "Garbage input",
			email:       "fds-lkfjsdlk-fjs",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := RequestHeaders{}
			err := headers.setFrom(tt.email)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, headers.From.Name, tt.expected.From.Name)
			assert.Equal(t, headers.From.Address, tt.expected.From.Address)
		})
	}
}

func TestCommentExtractor_extract(t *testing.T) {
	tests := []struct {
		name            string
		tokens          string
		index           int
		expectedComment string
		expectedNext    int
		expectError     bool
	}{
		{
			name:            "Standard example",
			tokens:          "(foo (bar)) Apache/1.0",
			index:           0,
			expectedComment: "(foo (bar))",
			expectedNext:    12,
			expectError:     false,
		},
		{
			name:            "Standard example ending a token sequence",
			tokens:          "Apache/1.0 (foo (bar) baz)",
			index:           11,
			expectedComment: "(foo (bar) baz)",
			expectedNext:    26,
			expectError:     false,
		},
		{
			name:            "Example preceeded by immediate token",
			tokens:          "(foo)baz",
			index:           0,
			expectedComment: "(foo)",
			expectedNext:    5,
			expectError:     false,
		},
		{
			name:            "Example preceeded by LWS",
			tokens:          "((foo))\r\n \t bar",
			index:           0,
			expectedComment: "((foo))",
			expectedNext:    12,
			expectError:     false,
		},
		{
			name:            "Example ending with LWS",
			tokens:          "Test (ending (comm(ent))) \r\n\t",
			index:           5,
			expectedComment: "(ending (comm(ent)))",
			expectedNext:    29,
			expectError:     false,
		},
		{
			name:        "Example not starting on valid comment",
			tokens:      "Foo Bar)",
			index:       4,
			expectError: true,
		},
		{
			name:        "Unclosed comment",
			tokens:      "Product (comment (never) ends",
			index:       8,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, next, err := commentExtractor(tt.tokens).extract(tt.index)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, c, tt.expectedComment)
			assert.Equal(t, next, tt.expectedNext)
		})
	}
}

func TestProductVersionExtractor_extract(t *testing.T) {
	tests := []struct {
		name                 string
		tokens               string
		index                int
		expectedProductToken string
		expectedNext         int
	}{
		{
			name:                 "Standard example",
			tokens:               "Apache/1.0 (foo (bar))",
			index:                0,
			expectedProductToken: "Apache/1.0",
			expectedNext:         11,
		},
		{
			name:                 "Standard example ending a token sequence",
			tokens:               "(foo (bar) baz) Foo/1.2.3",
			index:                16,
			expectedProductToken: "Foo/1.2.3",
			expectedNext:         25,
		},
		{
			name:                 "Example preceeded by immediate comment",
			tokens:               "Token(comment)",
			index:                0,
			expectedProductToken: "Token",
			expectedNext:         5,
		},
		{
			name:                 "Example preceeded by LWS",
			tokens:               "CERN-LineMode/2.15\r\n \t bar",
			index:                0,
			expectedProductToken: "CERN-LineMode/2.15",
			expectedNext:         23,
		},
		{
			name:                 "Example ending with LWS",
			tokens:               "Test  \r\n\t",
			index:                0,
			expectedProductToken: "Test",
			expectedNext:         9,
		},
		{
			name:                 "Empty product token",
			tokens:               "Test  (wow)",
			index:                4,
			expectedProductToken: "",
			expectedNext:         6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, next := productVersionExtractor(tt.tokens).extract(tt.index)
			assert.Equal(t, c, tt.expectedProductToken)
			assert.Equal(t, next, tt.expectedNext)
		})
	}
}

func TestProductVersionParser_parse(t *testing.T) {
	tests := []struct {
		name         string
		productToken string
		expected     ProductVersion
		expectError  bool
	}{
		{
			name:         "Standard product token",
			productToken: "Apache",
			expected: ProductVersion{
				Product: "Apache",
				Version: "",
			},
			expectError: false,
		},
		{
			name:         "More complex product token",
			productToken: "libwww/2.17be",
			expected: ProductVersion{
				Product: "libwww",
				Version: "2.17be",
			},
			expectError: false,
		},
		{
			name:         "Product token with valid punctuation",
			productToken: "lib_http-client/1.0.0-beta_2",
			expected: ProductVersion{
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
			res, err := productVersionParser(tt.productToken).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res.Product, tt.expected.Product)
			assert.Equal(t, res.Version, tt.expected.Version)
		})
	}
}

func TestRequestHeaders_setUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		string      string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:   "Classic example",
			string: "Mozilla/5.0",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{},
					Products: []ProductVersion{
						{Product: "Mozilla", Version: "5.0"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Multiple products",
			string: "Mozilla/5.0 Gecko/20100101 Firefox/115.0",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{},
					Products: []ProductVersion{
						{Product: "Mozilla", Version: "5.0"},
						{Product: "Gecko", Version: "20100101"},
						{Product: "Firefox", Version: "115.0"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Product followed by comment",
			string: "curl/7.88.1 (x86_64-pc-linux-gnu)",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(x86_64-pc-linux-gnu)",
					},
					Products: []ProductVersion{
						{Product: "curl", Version: "7.88.1"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Comment followed by product",
			string: "(x11; linux x86_64) curl/8.1.0",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(x11; linux x86_64)",
					},
					Products: []ProductVersion{
						{Product: "curl", Version: "8.1.0"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Interleaved product/comment/product",
			string: "MyAgent/1.0 (compatible) EngineX/2.4",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(compatible)",
					},
					Products: []ProductVersion{
						{Product: "MyAgent", Version: "1.0"},
						{Product: "EngineX", Version: "2.4"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Multiple adjacent comments",
			string: "Foo/1.2 (alpha)(beta)(rc1)",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(alpha)",
						"(beta)",
						"(rc1)",
					},
					Products: []ProductVersion{
						{Product: "Foo", Version: "1.2"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "No whitespace",
			string: "A/1(B)(C)D/2",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(B)",
						"(C)",
					},
					Products: []ProductVersion{
						{Product: "A", Version: "1"},
						{Product: "D", Version: "2"},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "Bizarre yet valid input",
			string: "lib_http-client/1.0.0-beta_2\r\n  \t (build+20240201)\t(A)()   ",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(build+20240201)",
						"(A)",
						"()",
					},
					Products: []ProductVersion{
						{Product: "lib_http-client", Version: "1.0.0-beta_2"},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "Unclosed comment",
			string:      "Mozilla/5.0 (X11 Linux",
			expectError: true,
		},
		{
			name:        "Unopened comment",
			string:      "Mozilla/5.0 X11)",
			expectError: true,
		},
		{
			name:        "Invalid product",
			string:      "/1.0",
			expectError: true,
		},
		{
			name:        "Anoter invalid product",
			string:      "Foo/",
			expectError: true,
		},
		{
			name:   "The Behemoth",
			string: "A/1 (a(b(c(d(e))))) B/2 (x)(y(z))",
			expected: RequestHeaders{
				UserAgent: UserAgent{
					Comments: []string{
						"(a(b(c(d(e)))))",
						"(x)",
						"(y(z))",
					},
					Products: []ProductVersion{
						{Product: "A", Version: "1"},
						{Product: "B", Version: "2"},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := RequestHeaders{}
			err := headers.setUserAgent(tt.string)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, headers.UserAgent.Comments, tt.expected.UserAgent.Comments)
			expectedProducts := tt.expected.UserAgent.Products
			actualProducts := headers.UserAgent.Products

			if len(expectedProducts) != len(actualProducts) {
				t.Errorf("different sizes. got: (%v, len: %d), want: (%v, len: %d)", actualProducts, len(actualProducts), expectedProducts, len(expectedProducts))
				return
			}

			for i := range len(actualProducts) {
				assert.Equal(t, actualProducts[i].Product, expectedProducts[i].Product)
				assert.Equal(t, actualProducts[i].Version, expectedProducts[i].Version)
			}
		})
	}
}

func TestRequestHeaders_setAllow(t *testing.T) {
	tests := []struct {
		name        string
		string      string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:   "Single method",
			string: "GET",
			expected: RequestHeaders{
				Allow: []Method{"GET"},
			},
			expectError: false,
		},
		{
			name:   "Multiple methods, common form",
			string: "GET, POST, HEAD",
			expected: RequestHeaders{
				Allow: []Method{"GET", "POST", "HEAD"},
			},
			expectError: false,
		},
		{
			name:   "No whitespace",
			string: "GET,POST,PUT,HEAD",
			expected: RequestHeaders{
				Allow: []Method{"GET", "POST", "PUT", "HEAD"},
			},
			expectError: false,
		},
		{
			name:   "Mixed LWS",
			string: "get ,\r\n\tPost,HeAd",
			expected: RequestHeaders{
				Allow: []Method{"get", "Post", "HeAd"},
			},
			expectError: false,
		},
		{
			name:        "Empty method",
			string:      "GET,,POST",
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
			headers := RequestHeaders{}

			err := headers.setAllow(tt.string)
			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, headers.Allow, tt.expected.Allow)
		})
	}
}

func TestRequestHeaders_setContentEncoding(t *testing.T) {
	tests := []struct {
		name        string
		string      string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:   "Canonical value",
			string: "x-gzip",
			expected: RequestHeaders{
				ContentEncoding: "x-gzip",
			},
			expectError: false,
		},
		{
			name:   "Non-standard casing of x-gzip",
			string: "X-gZIp",
			expected: RequestHeaders{
				ContentEncoding: "x-gzip",
			},
			expectError: false,
		},
		{
			name:   "Non-standard casing of x-compress",
			string: "x-CoMprEss",
			expected: RequestHeaders{
				ContentEncoding: "x-compress",
			},
			expectError: false,
		},
		{
			name:   "Non-standard token",
			string: "compress2",
			expected: RequestHeaders{
				ContentEncoding: "compress2",
			},
			expectError: false,
		},
		{
			name:        "Contains LWS",
			string:      "x-gzip x-compress",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := RequestHeaders{}

			err := headers.setContentEncoding(tt.string)
			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, headers.ContentEncoding, tt.expected.ContentEncoding)
		})
	}
}

func TestRequestHeaders_setContentLength(t *testing.T) {
	tests := []struct {
		name        string
		string      string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:   "Zero length",
			string: "0",
			expected: RequestHeaders{
				ContentLength: 0,
			},
			expectError: false,
		},
		{
			name:   "Small positive integer",
			string: "256",
			expected: RequestHeaders{
				ContentLength: 256,
			},
			expectError: false,
		},
		{
			name:   "Leading zeros",
			string: "00042",
			expected: RequestHeaders{
				ContentLength: 42,
			},
			expectError: false,
		},
		{
			name:   "Maximum uint64",
			string: "18446744073709551615",
			expected: RequestHeaders{
				ContentLength: 18446744073709551615,
			},
			expectError: false,
		},
		{
			name:        "Non-digit character",
			string:      "1e5",
			expectError: true,
		},
		{
			name:        "Integer overflow",
			string:      "18446744073709551616",
			expectError: true,
		},
		{
			name:        "Negative integer",
			string:      "-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := RequestHeaders{}

			err := headers.setContentLength(tt.string)
			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, headers.ContentLength, tt.expected.ContentLength)
		})
	}
}

func TestRequestHeaders_setUnrecognized(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		expected    RequestHeaders
		expectError bool
	}{
		{
			name:  "Standard Example",
			key:   "Foo",
			value: "Bar Baz",
			expected: RequestHeaders{
				Unrecognized: map[string]string{
					"Foo": "Bar Baz",
				},
			},
			expectError: false,
		},
		{
			name:  "Empty value",
			key:   "X-Empty",
			value: "",
			expected: RequestHeaders{
				Unrecognized: map[string]string{
					"X-Empty": "",
				},
			},
			expectError: false,
		},
		{
			name:  "Value with tspecials and spaces",
			key:   "X-Weird",
			value: "foo/bar;baz=qux\r\n \ttest",
			expected: RequestHeaders{
				Unrecognized: map[string]string{
					"X-Weird": "foo/bar;baz=qux\r\n \ttest",
				},
			},
			expectError: false,
		},
		{
			name:        "Invalid control character",
			key:         "X-Bad",
			value:       "hello\x01world",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := RequestHeaders{}

			err := headers.setUnrecognized(tt.key, tt.value)
			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.MapEqual(t, headers.Unrecognized, tt.expected.Unrecognized)
		})
	}
}

func TestContentTypeParametersParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		parameters  string
		expected    map[string]string
		expectError bool
	}{
		{
			name:       "Single parameter",
			parameters: "charset=utf-8",
			expected: map[string]string{
				"charset": "utf-8",
			},
			expectError: false,
		},
		{
			name:       "Multiple parameters, common form",
			parameters: "charset=utf-8; boundary=abc123",
			expected: map[string]string{
				"charset":  "utf-8",
				"boundary": "abc123",
			},
			expectError: false,
		},
		{
			name:       "Extra LWS surrounding ; delimeter",
			parameters: "charset=utf-8\r\n ;\t\r\n\tboundary=abc123",
			expected: map[string]string{
				"charset":  "utf-8",
				"boundary": "abc123",
			},
			expectError: false,
		},
		{
			name:       "Quoted-string value",
			parameters: "charset=\"utf-8\"",
			expected: map[string]string{
				"charset": "utf-8",
			},
			expectError: false,
		},
		{
			name:       "Quoted-string containing semicolon",
			parameters: "boundary=\"foo;bar\"",
			expected: map[string]string{
				"boundary": "foo;bar",
			},
			expectError: false,
		},
		{
			name:       "Quoted-string containing equal sign",
			parameters: "boundary=\"foo=bar=baz\"",
			expected: map[string]string{
				"boundary": "foo=bar=baz",
			},
			expectError: false,
		},
		{
			name:       "Mixed quoted and token parameters",
			parameters: "charset=utf-8\t;\r\n  boundary=\"abc;123=xyz\"",
			expected: map[string]string{
				"charset":  "utf-8",
				"boundary": "abc;123=xyz",
			},
			expectError: false,
		},
		{
			name:       "Empty quoted-string value",
			parameters: "param=\"\"",
			expected: map[string]string{
				"param": "",
			},
			expectError: false,
		},
		{
			name:        "Missing attribute",
			parameters:  "=foo",
			expectError: true,
		},
		{
			name:        "Missing value",
			parameters:  "charset=",
			expectError: true,
		},
		{
			name:        "Missing equal sign",
			parameters:  "test",
			expectError: true,
		},
		{
			name:        "Empty string",
			parameters:  "",
			expectError: true,
		},
		{
			name:        "Unterminated quoted-string",
			parameters:  "boundary=\"abc",
			expectError: true,
		},
		{
			name:       "The Beast",
			parameters: "a=b ; c=\"d;e=f;g\"\t;\r\n\th=i",
			expected: map[string]string{
				"a": "b",
				"c": "d;e=f;g",
				"h": "i",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := contentTypeParametersParser(tt.parameters).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.MapEqual(t, res, tt.expected)
		})
	}
}

func TestContentTypeParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    ContentType
		expectError bool
	}{
		{
			name:        "Simple type/subtype",
			contentType: "text/plain",
			expected: ContentType{
				Type:    "text",
				Subtype: "plain",
			},
			expectError: false,
		},
		{
			name:        "Type/subtype with charset",
			contentType: "text/html; charset=utf-8",
			expected: ContentType{
				Type:    "text",
				Subtype: "html",
				Parameters: map[string]string{
					"charset": "utf-8",
				},
			},
			expectError: false,
		},
		{
			name:        "Multiple parameters",
			contentType: "multipart/form-data; boundary=abc123; charset=utf-8",
			expected: ContentType{
				Type:    "multipart",
				Subtype: "form-data",
				Parameters: map[string]string{
					"boundary": "abc123",
					"charset":  "utf-8",
				},
			},
			expectError: false,
		},
		{
			name:        "Extra whitespace",
			contentType: " \t application/json \r\n\t ;  charset=utf-8",
			expected: ContentType{
				Type:    "application",
				Subtype: "json",
				Parameters: map[string]string{
					"charset": "utf-8",
				},
			},
			expectError: false,
		},
		{
			name:        "Complex subtype",
			contentType: "application/vnd.mycompany.mytype+json",
			expected: ContentType{
				Type:    "application",
				Subtype: "vnd.mycompany.mytype+json",
			},
			expectError: false,
		},
		{
			name:        "Missing subtype",
			contentType: "text",
			expectError: true,
		},
		{
			name:        "Empty parameter section",
			contentType: "text/plain;",
			expectError: true,
		},
		{
			name:        "Invalid token in type",
			contentType: "text@html/plain",
			expectError: true,
		},
		{
			name:        "No whitespace",
			contentType: "application/xml;charset=\"utf-8\";version=1.0",
			expected: ContentType{
				Type:    "application",
				Subtype: "xml",
				Parameters: map[string]string{
					"charset": "utf-8",
					"version": "1.0",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := contentTypeParser(tt.contentType).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res.Type, tt.expected.Type)
			assert.Equal(t, res.Subtype, tt.expected.Subtype)
			assert.MapEqual(t, res.Parameters, tt.expected.Parameters)
		})
	}
}
