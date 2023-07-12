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
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// CycleState is a WebAssembly implementation of framework.CycleState.
//
// # Notes
//
//   - Values stored by one plugin cannot be read, altered, or deleted by
//     another plugin.
//   - See /RATIONALE.md and /guest/RATIONALE.md for design details.
type CycleState interface {
	// Read retrieves data with the given "key" or returns false.
	Read(key string) (any, bool)

	// Write stores the given "val" in CycleState with the given "key".
	Write(key string, val any)

	// Delete deletes data with the given key.
	Delete(key string)
}

// Plugin is a WebAssembly implementation of framework.Plugin.
type Plugin interface {
	// This doesn't define `Name() string`. See /RATIONALE.md for impact
}

// PreFilterPlugin is a WebAssembly implementation of
// framework.PreFilterPlugin. When non-nil, the `nodeNames` result contains a
// unique set of node names to process.
//
// # Notes
//
//   - Any state kept in the plugin should be assigned to CycleState, not
//     global variables.
//   - Duplicate nodeNames are a bug, but will not cause a failure.
type PreFilterPlugin interface {
	Plugin

	PreFilter(state CycleState, pod Pod) (nodeNames []string, status *Status)
}

// FilterPlugin is a WebAssembly implementation of framework.FilterPlugin.
type FilterPlugin interface {
	Plugin

	Filter(state CycleState, pod Pod, nodeInfo NodeInfo) *Status
}

// ScorePlugin is a WebAssembly implementation of framework.ScorePlugin.
//
// Note: This is int32, not int64. See /RATIONALE.md for why.
type ScorePlugin interface {
	Plugin

	Score(state CycleState, pod Pod, nodeName string) (int32, *Status)
}

type NodeInfo interface {
	Node() *protoapi.Node
}

type Pod interface {
	Metadata() *meta.ObjectMeta
	Spec() *protoapi.PodSpec
	Status() *protoapi.PodStatus
}
