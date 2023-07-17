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
