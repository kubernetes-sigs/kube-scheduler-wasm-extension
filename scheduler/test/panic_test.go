package test

import (
	"errors"
	"testing"
)

// TestCapturePanic was originally adapted from wazero require.TestCapturePanic
func TestCapturePanic(t *testing.T) {
	tests := []struct {
		name     string
		panics   func()
		expected string
	}{
		{
			name:     "doesn't panic",
			panics:   func() {},
			expected: "",
		},
		{
			name:     "panics with error",
			panics:   func() { panic(errors.New("error")) },
			expected: "error",
		},
		{
			name:     "panics with string",
			panics:   func() { panic("crash") },
			expected: "crash",
		},
		{
			name:     "panics with object",
			panics:   func() { panic(struct{}{}) },
			expected: "{}",
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			captured := CapturePanic(tc.panics)
			if captured != tc.expected {
				t.Fatalf("expected %s, but found %s", tc.expected, captured)
			}
		})
	}
}
