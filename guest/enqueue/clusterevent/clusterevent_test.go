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

package clusterevent

import (
	"bytes"
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

func TestEncodeClusterEvents(t *testing.T) {
	tests := []struct {
		name     string
		input    []api.ClusterEvent
		expected []byte
	}{
		{
			name: "nil -> nil",
		},
		{
			name:  "empty -> nil",
			input: make([]api.ClusterEvent, 0),
		},
		{
			name: "Resource+ActionType",
			// From https://github.com/kubernetes-sigs/kube-scheduler-simulator/blob/06846c8c8313d3bd4f0fcbd4e533fb3d2f8375a1/simulator/docs/sample/nodenumber/plugin.go#L79C3-L79C56
			input: []api.ClusterEvent{{Resource: api.Node, ActionType: api.Add}},
			expected: []byte{
				byte(api.Node), 0, 0, 0,
				byte(api.Add), 0, 0, 0,
			},
		},
		{
			name: "multiple flags", // Note: currently, all flags fit into a single byte.
			input: []api.ClusterEvent{
				{Resource: api.Node, ActionType: api.UpdateNodeLabel | api.UpdateNodeCondition},
			},
			expected: []byte{
				byte(api.Node), 0, 0, 0,
				byte(api.UpdateNodeLabel | api.UpdateNodeCondition), 0, 0, 0,
			},
		},
		{
			name: "two",
			input: []api.ClusterEvent{
				{Resource: api.Node, ActionType: api.Add},
				{Resource: api.PersistentVolume, ActionType: api.Delete},
			},
			expected: []byte{
				byte(api.Node), 0, 0, 0,
				byte(api.Add), 0, 0, 0,
				byte(api.PersistentVolume), 0, 0, 0,
				byte(api.Delete), 0, 0, 0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeClusterEvents(tc.input)
			if want, have := tc.expected, encoded; (want == nil && have != nil) || !bytes.Equal(want, have) {
				t.Fatalf("unexpected value: %v != %v", want, have)
			}
		})
	}
}
