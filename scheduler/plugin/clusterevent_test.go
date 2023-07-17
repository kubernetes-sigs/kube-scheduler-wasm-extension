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

package wasm

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func TestEncodeClusterEvents(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []framework.ClusterEvent
	}{
		{
			name: "nil -> nil",
		},
		{
			name:  "empty -> nil",
			input: make([]byte, 0),
		},
		{
			name: "skip only truncated",
			input: []byte{
				byte(gvkNode), 0, 0, 0,
				byte(framework.Add), 0, 0, // not u32le
			},
		},
		{
			name: "Resource+ActionType",
			input: []byte{
				byte(gvkNode), 0, 0, 0,
				byte(framework.Add), 0, 0, 0,
				0, // no label is only the terminator
			},
			// From https://github.com/kubernetes-sigs/kube-scheduler-simulator/blob/06846c8c8313d3bd4f0fcbd4e533fb3d2f8375a1/simulator/docs/sample/nodenumber/plugin.go#L79C3-L79C56
			expected: []framework.ClusterEvent{{Resource: framework.Node, ActionType: framework.Add}},
		},
		{
			name: "multiple flags", // Note: currently, all flags fit into a single byte.
			input: []byte{
				byte(gvkNode), 0, 0, 0,
				byte(framework.UpdateNodeLabel | framework.UpdateNodeCondition), 0, 0, 0,
			},
			expected: []framework.ClusterEvent{
				{Resource: framework.Node, ActionType: framework.UpdateNodeLabel | framework.UpdateNodeCondition},
			},
		},
		{
			name: "two",
			input: []byte{
				byte(gvkNode), 0, 0, 0,
				byte(framework.Add), 0, 0, 0,
				byte(gvkPersistentVolume), 0, 0, 0,
				byte(framework.Delete), 0, 0, 0,
			},
			expected: []framework.ClusterEvent{
				{Resource: framework.Node, ActionType: framework.Add},
				{Resource: framework.PersistentVolume, ActionType: framework.Delete},
			},
		},
		{
			name: "skip truncated last",
			input: []byte{
				byte(gvkNode), 0, 0, 0,
				byte(framework.Add), 0, 0, 0,
				byte(gvkPersistentVolume), 0, 0, 0,
				byte(framework.Delete), 0, 0, // not u32le
			},
			expected: []framework.ClusterEvent{
				{Resource: framework.Node, ActionType: framework.Add},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := decodeClusterEvents(tc.input)
			if want, have := tc.expected, encoded; (want == nil && have != nil) || !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected value: %v != %v", want, have)
			}
		})
	}
}
