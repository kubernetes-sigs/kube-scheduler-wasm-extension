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
	"encoding/binary"
	"strconv"

	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// gvk are the framework.GVK defined in the Wasm ABI.
type gvk uint32

// Constants for GVKs.
const (
	gvkPod gvk = iota
	gvkNode
	gvkPersistentVolume
	gvkPersistentVolumeClaim
	gvkPodSchedulingContext
	gvkResourceClaim
	gvkStorageClass
	gvkCSINode
	gvkCSIDriver
	gvkCSIStorageCapacity
	gvkWildCard
)

func (gvk gvk) toGVK() framework.GVK {
	if int(gvk) < len(gvkToGVK) {
		return gvkToGVK[gvk]
	}
	return framework.GVK("GVK(" + strconv.Itoa(int(gvk)) + ")")
}

var gvkToGVK = [...]framework.GVK{
	"Pod",
	"Node",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"PodSchedulingContext",
	"ResourceClaim",
	"storage.k8s.io/StorageClass",
	"storage.k8s.io/CSINode",
	"storage.k8s.io/CSIDriver",
	"storage.k8s.io/CSIStorageCapacity",
	"*",
}

// sizeEncodedClusterEvent is the size in bytes to encode
// framework.ClusterEvent with 32-bit little endian gvk and ActionType
const sizeEncodedClusterEvent = 4 + 4

func decodeClusterEvents(b []byte) (clusterEvents []framework.ClusterEvent) {
	for i := 0; i+sizeEncodedClusterEvent <= len(b); i += sizeEncodedClusterEvent {
		clusterEvents = append(clusterEvents, framework.ClusterEvent{
			Resource:   gvk(binary.LittleEndian.Uint32(b[i:])).toGVK(),
			ActionType: framework.ActionType(binary.LittleEndian.Uint32(b[i+4:])),
		})
	}
	return
}
