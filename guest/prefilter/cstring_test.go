/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package prefilter

import (
	"bytes"
	"testing"
)

func TestToNULTerminated(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []byte
	}{
		{
			name: "nil -> nil",
		},
		{
			name:  "empty -> nil",
			input: make([]string, 0),
		},
		{
			name:     "one",
			input:    []string{"a"},
			expected: []byte{'a', 0},
		},
		{
			name:     "two",
			input:    []string{"a", "two"},
			expected: []byte{'a', 0, 't', 'w', 'o', 0},
		},
		{
			name:     "skip empty",
			input:    []string{"a", "", "two"},
			expected: []byte{'a', 0, 't', 'w', 'o', 0},
		},
		{
			name:     "unicode",
			input:    []string{"a", "fóo", "c"},
			expected: []byte{'a', 0, 'f', 0xc3, 0xb3, 'o', 0, 'c', 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cstring := toNULTerminated(tc.input)
			if want, have := tc.expected, cstring; (want == nil && have != nil) || !bytes.Equal(want, have) {
				t.Fatalf("unexpected cstring: %v != %v", want, have)
			}
		})
	}
}
