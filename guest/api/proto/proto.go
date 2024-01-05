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

// Package proto includes any types derived from Kubernetes protobuf messages.
package proto

import api "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"

type KObject interface {
	Metadata

	GetKind() string
	GetApiVersion() string
}

// Metadata are fields on top-level types, used for logging and metrics.
type Metadata interface {
	GetUid() string
	GetName() string
	GetNamespace() string
	GetResourceVersion() string
}

type Node interface {
	Metadata

	Spec() *api.NodeSpec
	Status() *api.NodeStatus
	GetKind() string
	GetApiVersion() string
}

type NodeList interface {
	Items() []Node
}

type Pod interface {
	Metadata

	Spec() *api.PodSpec
	Status() *api.PodStatus
	GetKind() string
	GetApiVersion() string
}
