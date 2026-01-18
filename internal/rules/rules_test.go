package rules

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name     string
		rules    string
		expected []string
	}{
		{
			name:     "Standard rule set",
			rules:    "foo, bar, baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "Rule set with newline LWS",
			rules:    "\r\n\tfoo , \r\n \r\n\tbar, \tbaz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "LWS within words",
			rules:    "fo \r\n\t o,  \tb\t ar, ba\001z",
			expected: []string{"fo \r\n\t o", "b\t ar", "ba\001z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.SliceEqual(t, Extract(tt.rules), tt.expected)
		})
	}
}
