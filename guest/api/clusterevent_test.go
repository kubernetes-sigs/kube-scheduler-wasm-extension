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

package api

import (
	"testing"
)

func TestGVK_String(t *testing.T) {
	tests := []struct {
		name     string
		gvk      GVK
		expected string
	}{
		{name: "Pod", gvk: Pod, expected: "Pod"},
		{name: "Node", gvk: Node, expected: "Node"},
		{name: "PersistentVolume", gvk: PersistentVolume, expected: "PersistentVolume"},
		{name: "PersistentVolumeClaim", gvk: PersistentVolumeClaim, expected: "PersistentVolumeClaim"},
		{name: "PodSchedulingContext", gvk: PodSchedulingContext, expected: "PodSchedulingContext"},
		{name: "ResourceClaim", gvk: ResourceClaim, expected: "ResourceClaim"},
		{name: "StorageClass", gvk: StorageClass, expected: "storage.k8s.io/StorageClass"},
		{name: "CSINode", gvk: CSINode, expected: "storage.k8s.io/CSINode"},
		{name: "CSIStorageCapacity", gvk: CSIStorageCapacity, expected: "storage.k8s.io/CSIStorageCapacity"},
		{name: "WildCard", gvk: WildCard, expected: "*"},
		{name: "undefined", gvk: 99, expected: "GVK(99)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.expected, tc.gvk.String(); want != have {
				t.Fatalf("unexpected string: %v != %v", want, have)
			}
		})
	}
}
