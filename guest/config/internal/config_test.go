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

package internal

import (
	"bytes"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{name: "empty"},
		{
			name:     "NUL terminated",
			input:    string([]byte{'a', 0, 't', 'w', 'o', 0}),
			expected: []byte{'a', 0, 't', 'w', 'o', 0},
		},
		{
			name:     "unicode",
			input:    "fóo",
			expected: []byte{'f', 0xc3, 0xb3, 'o'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configRead = false
			defer func() {
				configRead = false
			}()

			for _, fn := range []func() string{
				func() string {
					return tc.input
				},
				func() string {
					panic("should cache the first read")
				},
			} {
				if want, have := tc.expected, Get(fn); (want == nil && have != nil) || !bytes.Equal(want, have) {
					t.Fatalf("unexpected config: %v != %v", want, have)
				}
			}
		})
	}
}
