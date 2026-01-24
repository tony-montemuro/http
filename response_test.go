package http

import (
	"testing"

	"github.com/tony-montemuro/http/internal/assert"
)

type intSetterTest struct {
	name        string
	value       int
	expected    int
	expectError bool
}

func TestSetStatus(t *testing.T) {
	tests := []intSetterTest{
		{
			name:        "Standard case",
			value:       200,
			expected:    200,
			expectError: false,
		},
		{
			name:        "Invaid status",
			value:       800,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := ResponseWriter{}
			err := rw.SetStatus(tt.value)

			ok := assert.ErrorStatus(t, err, tt.expectError)
			if !ok {
				return
			}

			assert.Equal(t, int(rw.response.code), tt.expected)
		})
	}
}
