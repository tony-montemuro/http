package http

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestEscapeSequence_unescape(t *testing.T) {
	tests := []struct {
		name        string
		arr         []byte
		index       int
		expected    byte
		expectError bool
	}{
		{
			name:        "Basic example",
			arr:         []byte("%3Ftest"),
			index:       0,
			expected:    '?',
			expectError: false,
		},
		{
			name:        "End of array",
			arr:         []byte("test%ad"),
			index:       4,
			expected:    173,
			expectError: false,
		},
		{
			name:        "Malformed escape sequence",
			arr:         []byte("Te%1jst"),
			index:       2,
			expectError: true,
		},
		{
			name:        "Truncated escape sequence",
			arr:         []byte("Test%"),
			index:       4,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := escapeSequence(tt.arr).unescape(tt.index)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res, tt.expected)
		})
	}
}

func AbsoluteUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         []byte
		expected    AbsoluteUri
		expectError bool
	}{
		{
			name: "Basic HTTP URL",
			uri:  []byte("http://www.w3.org/pub/WWW/TheProject.html"),
			expected: AbsoluteUri{
				Scheme: []byte("http"),
				Path:   []byte("//www.w3.org/pub/WWW/TheProject.html"),
			},
			expectError: false,
		},
		{
			name:        "Invalid empty scheme",
			uri:         []byte(":path/to/resource"),
			expectError: true,
		},
		{
			name:        "Invalid missing colon",
			uri:         []byte("http-www.example.com"),
			expectError: true,
		},
		{
			name: "Empty path after scheme",
			uri:  []byte("http:"),
			expected: AbsoluteUri{
				Scheme: []byte("http"),
				Path:   []byte(""), // or nil, depending on implementation preferences
			},
			expectError: false,
		},
		{
			name: "Complex valid scheme chars",
			uri:  []byte("soap.beep-1+2://api"),
			expected: AbsoluteUri{
				Scheme: []byte("soap.beep-1+2"),
				Path:   []byte("//api"),
			},
			expectError: false,
		},
		{
			name:        "Invalid underscore in scheme",
			uri:         []byte("my_scheme:data"),
			expectError: true,
		},
		{
			name: "Opaque URN",
			uri:  []byte("news:comp.infosystems.www.servers.unix"),
			expected: AbsoluteUri{
				Scheme: []byte("news"),
				Path:   []byte("comp.infosystems.www.servers.unix"),
			},
			expectError: false,
		},
		{
			name: "Reserved chars in path",
			uri:  []byte("mailto:user@example.com?subject=Hello"),
			expected: AbsoluteUri{
				Scheme: []byte("mailto"),
				Path:   []byte("user@example.com?subject=Hello"),
			},
			expectError: false,
		},
		{
			name:        "Invalid space in path",
			uri:         []byte("file:documents/my file.txt"),
			expectError: true,
		},
		{
			name:        "Invalid fragment in absoluteURI",
			uri:         []byte("http://example.com#heading1"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := absoluteUriParser(tt.uri).Parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, res.Path, tt.expected.Path)
			assert.SliceEqual(t, res.Scheme, tt.expected.Scheme)
		})
	}
}

func RelativeUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         []byte
		expected    RelativeUri
		expectError bool
	}{
		{
			name: "Basic relative path",
			uri:  []byte("images/logo.png"),
			expected: RelativeUri{
				Path: []byte("images/logo.png"),
			},
			expectError: false,
		},
		{
			name: "Absolute path",
			uri:  []byte("/usr/local/bin"),
			expected: RelativeUri{
				Path: []byte("/usr/local/bin"),
			},
			expectError: false,
		},
		{
			name: "Network path with resource",
			uri:  []byte("//example.com/index.html"),
			expected: RelativeUri{
				NetLoc: []byte("example.com"),
				Path:   []byte("/index.html"),
			},
			expectError: false,
		},
		{
			name: "Path with multiple parameters",
			uri:  []byte("library;version=2;format=json"),
			expected: RelativeUri{
				Path: []byte("library"),
				Params: [][]byte{
					[]byte("version=2"),
					[]byte("format=json"),
				},
			},
			expectError: false,
		},
		{
			name: "Query string only",
			uri:  []byte("?search=test&page=1"),
			expected: RelativeUri{
				Query: []byte("search=test&page=1"),
			},
			expectError: false,
		},
		{
			name: "Full URI components",
			uri:  []byte("//api.srv/v1/user;auth=oauth?debug=true"),
			expected: RelativeUri{
				NetLoc: []byte("api.srv"),
				Path:   []byte("/v1/user"),
				Params: [][]byte{
					[]byte("auth=oauth"),
				},
				Query: []byte("debug=true"),
			},
			expectError: false,
		},
		{
			name: "Parameters containing slashes",
			uri:  []byte("file;type=application/pdf"),
			expected: RelativeUri{
				Path: []byte("file"),
				Params: [][]byte{
					[]byte("type=application/pdf"),
				},
			},
			expectError: false,
		},
		{
			name: "Path with valid special chars",
			uri:  []byte("user:pass@home+id"),
			expected: RelativeUri{
				Path: []byte("user:pass@home+id"),
			},
			expectError: false,
		},
		{
			name:        "Invalid fragment in relativeURI",
			uri:         []byte("/index#section1"),
			expectError: true,
		},
		{
			name:        "Invalid space in path",
			uri:         []byte("/my folder/file"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := relativeUriParser(tt.uri).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, res.NetLoc, tt.expected.NetLoc)
			assert.SliceEqual(t, res.Path, tt.expected.Path)
			assert.MatrixEqual(t, res.Params, tt.expected.Params)
			assert.SliceEqual(t, res.Query, tt.expected.Query)
		})
	}
}

type pathExpected struct {
	path   []byte
	params [][]byte
	query  []byte
}

func TestAbsPathUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         []byte
		expected    pathExpected
		expectError bool
	}{
		{
			name:        "Root path",
			uri:         []byte("/"),
			expected:    pathExpected{path: []byte{'/'}, params: [][]byte{{}}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no params or query",
			uri:         []byte("/info/document/1"),
			expected:    pathExpected{path: []byte("/info/document/1"), params: [][]byte{{}}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no query",
			uri:         []byte("/data;test/3;wow!"),
			expected:    pathExpected{path: []byte("/data"), params: [][]byte{[]byte("test/3"), []byte("wow!")}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no param",
			uri:         []byte("/foo/bar?test=3&t;est"),
			expected:    pathExpected{path: []byte("/foo/bar"), params: [][]byte{{}}, query: []byte("test=3&t;est")},
			expectError: false,
		},
		{
			name:        "Uri with no path",
			uri:         []byte("/;data/here?f00=bar"),
			expected:    pathExpected{path: []byte{'/'}, params: [][]byte{[]byte("data/here")}, query: []byte("f00=bar")},
			expectError: false,
		},
		{
			name:        "Uri with no params or path",
			uri:         []byte("/?;"),
			expected:    pathExpected{path: []byte{'/'}, params: [][]byte{{}}, query: []byte(";")},
			expectError: false,
		},
		{
			name:        "Uri with unsafe character",
			uri:         []byte("/te st/document/2"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, params, query, err := absPathUriParser(tt.uri).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, path, tt.expected.path)
			assert.MatrixEqual(t, params, tt.expected.params)
			assert.SliceEqual(t, query, tt.expected.query)
		})
	}
}

func TestRelPathUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         []byte
		expected    pathExpected
		expectError bool
	}{
		{
			name:        "Empty input",
			uri:         []byte(""),
			expected:    pathExpected{path: []byte{}, params: [][]byte{{}}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no params or query",
			uri:         []byte("info/document/1"),
			expected:    pathExpected{path: []byte("info/document/1"), params: [][]byte{{}}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no query",
			uri:         []byte("data;test/3;wow!"),
			expected:    pathExpected{path: []byte("data"), params: [][]byte{[]byte("test/3"), []byte("wow!")}, query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no param",
			uri:         []byte("foo/bar?test=3&t;est"),
			expected:    pathExpected{path: []byte("foo/bar"), params: [][]byte{{}}, query: []byte("test=3&t;est")},
			expectError: false,
		},
		{
			name:        "Uri with no path",
			uri:         []byte(";data/here?f00=bar"),
			expected:    pathExpected{path: []byte{}, params: [][]byte{[]byte("data/here")}, query: []byte("f00=bar")},
			expectError: false,
		},
		{
			name:        "Uri with no params or path",
			uri:         []byte("?;"),
			expected:    pathExpected{path: []byte{}, params: [][]byte{{}}, query: []byte(";")},
			expectError: false,
		},
		{
			name:        "Uri with unsafe character",
			uri:         []byte("te st/document/2"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, params, query, err := relPathUriParser(tt.uri).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, path, tt.expected.path)
			assert.MatrixEqual(t, params, tt.expected.params)
			assert.SliceEqual(t, query, tt.expected.query)
		})
	}
}

func TestUriPathParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		path        []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "Root path",
			path:        []byte{},
			expected:    []byte{},
			expectError: false,
		},
		{
			name:        "Single path",
			path:        []byte("info"),
			expected:    []byte("info"),
			expectError: false,
		},
		{
			name:        "Muli path",
			path:        []byte("info/document/1"),
			expected:    []byte("info/document/1"),
			expectError: false,
		},
		{
			name:        "Empty first segment",
			path:        []byte("/test"),
			expectError: true,
		},
		{
			name:        "Escaped path",
			path:        []byte("info/%7Btest%7D"),
			expected:    []byte("info/{test}"),
			expectError: false,
		},
		{
			name:        "Non-hex escape",
			path:        []byte("info/te%XDst"),
			expectError: true,
		},
		{
			name:        "Trimmed escape",
			path:        []byte("info/test%1"),
			expectError: true,
		},
		{
			name:        "Invalid characters path",
			path:        []byte("in;fo/te?st"),
			expectError: true,
		},
		{
			name:        "Strange but valid path",
			path:        []byte("test//a//"),
			expected:    []byte("test//a//"),
			expectError: false,
		},
		{
			name:        "Escaped space",
			path:        []byte("foo%20test"),
			expectError: true,
		},
		{
			name:        "Escaped delete",
			path:        []byte("foo%7Fbar"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := uriPathParser(tt.path).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestUriParamsParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		params      []byte
		expected    [][]byte
		expectError bool
	}{
		{
			name:        "No params",
			params:      []byte{},
			expected:    [][]byte{{}},
			expectError: false,
		},
		{
			name:        "Single param (;data/test)",
			params:      []byte("data/test"),
			expected:    [][]byte{[]byte("data/test")},
			expectError: false,
		},
		{
			name:        "Muli param (;data/test;stuff;1)",
			params:      []byte("data/test;stuff;1"),
			expected:    [][]byte{[]byte("data/test"), []byte("stuff"), []byte("1")},
			expectError: false,
		},
		{
			name:        "Escaped param (;info;{test})",
			params:      []byte("info;%7Btest%7D"),
			expected:    [][]byte{[]byte("info"), []byte("{test}")},
			expectError: false,
		},
		{
			name:        "Non-hex escape param (;info;te%XDst)",
			params:      []byte("info;te%XDst"),
			expectError: true,
		},
		{
			name:        "Trimmed escape (;info;test%1)",
			params:      []byte("info;test%1"),
			expectError: true,
		},
		{
			name:        "Invalid characters param (;in#fo;te>st)",
			params:      []byte("in#fo;te>st"),
			expectError: true,
		},
		{
			name:        "Strange but valid params (;;test;;a;;)",
			params:      []byte(";test;;a;;"),
			expected:    [][]byte{[]byte(""), []byte("test"), []byte(""), []byte("a"), []byte(""), []byte("")},
			expectError: false,
		},
		{
			name:        "Escaped CR in param (;foo;bar%0Dbaz)",
			params:      []byte(";foo;bar%0Dbaz"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := uriParamsParser(tt.params).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.MatrixEqual(t, res, tt.expected)
		})
	}
}

func TestUriQueryParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		query       []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "No query",
			query:       []byte{},
			expected:    []byte{},
			expectError: false,
		},
		{
			name:        "Basic query (?test)",
			query:       []byte("test"),
			expected:    []byte("test"),
			expectError: false,
		},
		{
			name:        "Mix of valid characters query (?test=3&foo!='bar')",
			query:       []byte("?test=3&foo!='bar'"),
			expected:    []byte("?test=3&foo!='bar'"),
			expectError: false,
		},
		{
			name:        "Escaped query (?info={test})",
			query:       []byte("info=%7Btest%7D"),
			expected:    []byte("info={test}"),
			expectError: false,
		},
		{
			name:        "Non-hex escape param (?info=te%XDst)",
			query:       []byte("info=te%XDst"),
			expectError: true,
		},
		{
			name:        "Trimmed escape (?info=test%1)",
			query:       []byte("info=test%1"),
			expectError: true,
		},
		{
			name:        "Invalid characters param (?in fo=te<st)",
			query:       []byte("in fo=te<st"),
			expectError: true,
		},
		{
			name:        "Escaped NUL param (?foo=x%20)",
			query:       []byte("/foo=x%20"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := uriQueryParser(tt.query).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.SliceEqual(t, res, tt.expected)
		})
	}
}

func TestSafeUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expected    string
		expectError bool
	}{
		{
			name:        "Safe URI",
			uri:         "https://smbelite.com",
			expected:    "https://smbelite.com",
			expectError: false,
		},
		{
			name:        "Not safe URI",
			uri:         "bad\turi",
			expectError: true,
		},
		{
			name:        "Safe URI with pound",
			uri:         "foo#bar",
			expected:    "foo#bar",
			expectError: false,
		},
		{
			name:        "Safe URI with ampersand",
			uri:         "foo%3Abar",
			expected:    "foo:bar",
			expectError: false,
		},
		{
			name:        "Unsafe URI from escaped contorl character",
			uri:         "foo%13bar",
			expectError: true,
		},
		{
			name:        "Unsafe URI from escaped unsafe character",
			uri:         "foo%3Cbar",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := safeUriParser(tt.uri).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, res, tt.expected)
		})
	}
}
