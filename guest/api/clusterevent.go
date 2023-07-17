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

import "strconv"

// ActionType is an integer to represent one type of resource change.
// Different ActionTypes can be bit-wised to compose new semantics.
type ActionType uint32

// Constants for ActionTypes.
const (
	Add    ActionType = 1 << iota // 1
	Delete                        // 10
	// UpdateNodeXYZ is only applicable for Node events.
	UpdateNodeAllocatable // 100
	UpdateNodeLabel       // 1000
	UpdateNodeTaint       // 10000
	UpdateNodeCondition   // 100000

	All ActionType = 1<<iota - 1 // 111111

	// Use the general Update type if you don't either know or care the specific sub-Update type to use.
	Update = UpdateNodeAllocatable | UpdateNodeLabel | UpdateNodeTaint | UpdateNodeCondition
)

// GVK is short for group/version/kind, which can uniquely represent a particular API resource.
type GVK uint32

// Constants for GVKs.
const (
	Pod GVK = iota
	Node
	PersistentVolume
	PersistentVolumeClaim
	PodSchedulingContext
	ResourceClaim
	StorageClass
	CSINode
	CSIDriver
	CSIStorageCapacity
	WildCard
)

// String implements fmt.Stringer
func (gvk GVK) String() string {
	if int(gvk) < len(gvkToString) {
		return gvkToString[gvk]
	}
	return "GVK(" + strconv.Itoa(int(gvk)) + ")"
}

var gvkToString = [...]string{
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

// ClusterEvent abstracts how a system resource's state gets changed.
// Resource represents the standard API resources such as Pod, Node, etc.
// ActionType denotes the specific change such as Add, Update or Delete.
type ClusterEvent struct {
	Resource   GVK
	ActionType ActionType
}

// ^-- Note: This does not include Label.
// See https://kubernetes.slack.com/archives/C09TP78DV/p1689409183711429

// IsWildCard returns true if ClusterEvent follows WildCard semantics
func (ce ClusterEvent) IsWildCard() bool {
	return ce.Resource == WildCard && ce.ActionType == All
}
