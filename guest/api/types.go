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
	PreFilter(state CycleState, pod Pod) (nodeNames []string, status *Status)
}

var _ PreFilterPlugin = PreFilterFunc(nil)

// PreFilterFunc adapts an ordinary function to a PreFilterPlugin.
type PreFilterFunc func(state CycleState, pod Pod) (nodeNames []string, status *Status)

// PreFilter returns f(state, pod).
func (f PreFilterFunc) PreFilter(state CycleState, pod Pod) (nodeNames []string, status *Status) {
	return f(state, pod)
}

// FilterPlugin is a WebAssembly implementation of framework.FilterPlugin.
type FilterPlugin interface {
	Filter(state CycleState, pod Pod, nodeInfo NodeInfo) *Status
}

var _ FilterPlugin = FilterFunc(nil)

// FilterFunc adapts an ordinary function to a FilterPlugin.
type FilterFunc func(state CycleState, pod Pod, nodeInfo NodeInfo) *Status

// Filter returns f(state, pod, nodeInfo).
func (f FilterFunc) Filter(state CycleState, pod Pod, nodeInfo NodeInfo) *Status {
	return f(state, pod, nodeInfo)
}

// ScorePlugin is a WebAssembly implementation of framework.ScorePlugin.
//
// Note: This is int32, not int64. See /RATIONALE.md for why.
type ScorePlugin interface {
	Score(state CycleState, pod Pod, nodeName string) (int32, *Status)
}

var _ ScorePlugin = ScoreFunc(nil)

// ScoreFunc adapts an ordinary function to a ScorePlugin.
type ScoreFunc func(state CycleState, pod Pod, nodeName string) (int32, *Status)

// Score returns f(pod, nodeName).
func (f ScoreFunc) Score(state CycleState, pod Pod, nodeName string) (int32, *Status) {
	return f(state, pod, nodeName)
}

type NodeInfo interface {
	Node() *protoapi.Node
}

type Pod interface {
	Metadata() *meta.ObjectMeta
	Spec() *protoapi.PodSpec
	Status() *protoapi.PodStatus
}
