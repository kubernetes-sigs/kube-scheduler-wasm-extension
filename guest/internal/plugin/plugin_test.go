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

package plugin

import (
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

type valPlugin string

type pointerPlugin struct {
	name string
}

func Test_set(t *testing.T) {
	tests := []struct {
		name   string
		p1, p2 api.Plugin
	}{
		{
			name: "val",
			p1:   valPlugin("a"),
			p2:   valPlugin("b"),
		},
		{
			name: "pointer",
			p1:   &pointerPlugin{"a"},
			p2:   &pointerPlugin{"b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() { current = nil }()

			MustSet(tc.p1)
			// Ensure we can set to the same instance
			MustSet(tc.p1)

			// Test as a bool we cannot set a different value.
			//
			// We can't test MustSet because panics are not yet recoverable.
			// https://github.com/tinygo-org/tinygo/issues/2914
			if set(tc.p2) {
				t.Fatalf("expected to be unable to set a different plugin")
			}
		})
	}
}
