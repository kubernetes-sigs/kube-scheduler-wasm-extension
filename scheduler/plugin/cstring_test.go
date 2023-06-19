package wasm

import (
	"reflect"
	"testing"
)

func TestFromNULTerminated(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		input    []byte
	}{
		{
			name: "nil -> nil",
		},
		{
			name:  "empty -> nil",
			input: make([]byte, 0),
		},
		{
			name:     "one",
			input:    []byte{'a', 0},
			expected: []string{"a"},
		},
		{
			name:     "two",
			input:    []byte{'a', 0, 't', 'w', 'o', 0},
			expected: []string{"a", "two"},
		},
		{
			name:     "skip empty",
			input:    []byte{'a', 0, 0, 't', 'w', 'o', 0},
			expected: []string{"a", "two"},
		},
		{
			name:     "unicode",
			input:    []byte{'a', 0, 'f', 0xc3, 0xb3, 'o', 0, 'c', 0},
			expected: []string{"a", "fóo", "c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries := fromNULTerminated(tc.input)
			if want, have := tc.expected, entries; (want == nil && have != nil) || !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected entries: %v != %v", want, have)
			}
		})
	}
}
