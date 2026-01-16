package message

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

func TestAbsPathUriParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		uri         []byte
		expected    AbsPathUri
		expectError bool
	}{
		{
			name:        "Root uri (/)",
			uri:         []byte("/"),
			expected:    AbsPathUri{Path: [][]byte{{}}, Params: [][]byte{{}}, Query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no params or query (/info/document/1)",
			uri:         []byte("/info/document/1"),
			expected:    AbsPathUri{Path: [][]byte{[]byte("info"), []byte("document"), []byte("1")}, Params: [][]byte{{}}, Query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no query (/data;test/3;wow!)",
			uri:         []byte("/data;test/3;wow!"),
			expected:    AbsPathUri{Path: [][]byte{[]byte("data")}, Params: [][]byte{[]byte("test/3"), []byte("wow!")}, Query: []byte{}},
			expectError: false,
		},
		{
			name:        "Uri with no param (/foo/bar?test=3&t;est)",
			uri:         []byte("/foo/bar?test=3&t;est"),
			expected:    AbsPathUri{Path: [][]byte{[]byte("foo"), []byte("bar")}, Params: [][]byte{{}}, Query: []byte("test=3&t;est")},
			expectError: false,
		},
		{
			name:        "Uri with no path (/;data/here?f00=bar)",
			uri:         []byte("/;data/here?f00=bar"),
			expected:    AbsPathUri{Path: [][]byte{{}}, Params: [][]byte{[]byte("data/here")}, Query: []byte("f00=bar")},
			expectError: false,
		},
		{
			name:        "Uri with no params or path (/?;)",
			uri:         []byte("/?;"),
			expected:    AbsPathUri{Path: [][]byte{{}}, Params: [][]byte{{}}, Query: []byte(";")},
			expectError: false,
		},
		{
			name:        "Uri with unsafe character (/te st/document/2)",
			uri:         []byte("/te st/document/2"),
			expected:    AbsPathUri{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := absPathUriParser(tt.uri).parse()

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.MatrixEqual(t, res.Path, tt.expected.Path)
			assert.MatrixEqual(t, res.Params, tt.expected.Params)
			assert.SliceEqual(t, res.Query, tt.expected.Query)
		})

	}
}

func TestUriPathParser_parse(t *testing.T) {
	tests := []struct {
		name        string
		path        []byte
		expected    [][]byte
		expectError bool
	}{
		{
			name:        "Root path (/)",
			path:        []byte{},
			expected:    [][]byte{{}},
			expectError: false,
		},
		{
			name:        "Single path (/info)",
			path:        []byte("info"),
			expected:    [][]byte{[]byte("info")},
			expectError: false,
		},
		{
			name:        "Muli path (/info/document/1)",
			path:        []byte("info/document/1"),
			expected:    [][]byte{[]byte("info"), []byte("document"), []byte("1")},
			expectError: false,
		},
		{
			name:        "Empty first segment (//test)",
			path:        []byte("/test"),
			expected:    [][]byte{},
			expectError: true,
		},
		{
			name:        "Escaped path (/info/{test})",
			path:        []byte("info/%7Btest%7D"),
			expected:    [][]byte{[]byte("info"), []byte("{test}")},
			expectError: false,
		},
		{
			name:        "Non-hex escape (/info/te%XDst)",
			path:        []byte("info/te%XDst"),
			expectError: true,
		},
		{
			name:        "Trimmed escape (/info/test%1)",
			path:        []byte("info/test%1"),
			expectError: true,
		},
		{
			name:        "Invalid characters path (/in;fo/te?st)",
			path:        []byte("in;fo/te?st"),
			expectError: true,
		},
		{
			name:        "Strange but valid path (/test//a//)",
			path:        []byte("test//a//"),
			expected:    [][]byte{[]byte("test"), []byte(""), []byte("a"), []byte(""), []byte("")},
			expectError: false,
		},
		{
			name:        "Escaped space (/foo%20test)",
			path:        []byte("foo%20test"),
			expectError: true,
		},
		{
			name:        "Escaped delete (/foo%7Ftest)",
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

			assert.MatrixEqual(t, res, tt.expected)
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
