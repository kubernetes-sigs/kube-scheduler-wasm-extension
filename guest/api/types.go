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
	"time"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
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
//   - The pod parameter is lazy to avoid unmarshal overhead when unused.
type PreFilterPlugin interface {
	Plugin

	PreFilter(state CycleState, pod proto.Pod) (nodeNames []string, status *Status)
}

// FilterPlugin is a WebAssembly implementation of framework.FilterPlugin.
//
// Note: The pod and nodeInfo parameters are lazy to avoid unmarshal overhead
// when unused.
type FilterPlugin interface {
	Plugin

	Filter(state CycleState, pod proto.Pod, nodeInfo NodeInfo) *Status
}

// PostFilterPlugin is a WebAssembly implementation of framework.PostFilterPlugin.
type PostFilterPlugin interface {
	Plugin

	PostFilter(state CycleState, pod proto.Pod, filteredNodeStatusMap NodeToStatus) (nominatedNodeName string, nominatingMode NominatingMode, status *Status)
}

// NominatingMode is the Mode which is returned from PostFilter.
type NominatingMode int32

// These are predefined modes
const (
	ModeNoop NominatingMode = iota
	ModeOverride
)

// EnqueueExtensions is a WebAssembly implementation of framework.EnqueueExtensions.
type EnqueueExtensions interface {
	EventsToRegister() []ClusterEvent
}

// PreScorePlugin is a WebAssembly implementation of framework.PreScorePlugin.
//
// Note: The pod and nodeList parameters are lazy to avoid unmarshal overhead
// when unused.
type PreScorePlugin interface {
	Plugin

	PreScore(state CycleState, pod proto.Pod, nodeList proto.NodeList) *Status
}

// ScorePlugin is a WebAssembly implementation of framework.ScorePlugin.
//
// Note: This is int32, not int64. See /RATIONALE.md for why.
type ScorePlugin interface {
	Plugin

	Score(state CycleState, pod proto.Pod, nodeName string) (int32, *Status)
}

// ScoreExtensions is a WebAssembly implementation of framework.ScoreExtensions.
type ScoreExtensions interface {
	Plugin

	NormalizeScore(state CycleState, pod proto.Pod, scores NodeScore) (map[string]int, *Status)
}

// ReservePlugin is a WebAssembly implementation of framework.ReservePlugin.
type ReservePlugin interface {
	Plugin

	Reserve(state CycleState, p proto.Pod, nodeName string) *Status
	Unreserve(state CycleState, p proto.Pod, nodeName string)
}

// PermitPlugin is a WebAssembly implementation of framework.PermitPlugin.
type PermitPlugin interface {
	Plugin

	Permit(state CycleState, p proto.Pod, nodeName string) (*Status, time.Duration)
}

// PreBindPlugin is a WebAssembly implementation of framework.PreBindPlugin.
type PreBindPlugin interface {
	Plugin

	PreBind(state CycleState, pod proto.Pod, nodeName string) *Status
}

// BindPlugin is a WebAssembly implementation of framework.BindPlugin.
type BindPlugin interface {
	Plugin

	Bind(state CycleState, pod proto.Pod, nodeName string) *Status
}

// PostBindPlugin is a WebAssembly implementation of framework.PostBindPlugin.
type PostBindPlugin interface {
	Plugin

	PostBind(state CycleState, pod proto.Pod, nodeName string)
}

type NodeInfo interface {
	// Metadata is a convenience that triggers Get.
	proto.Metadata

	Node() proto.Node
}

// NodeToStatus contains which Node got which status during the scheduling cycle.
type NodeToStatus interface {
	// NodeToStatus returns a map
	// which is keyed by the node name and valued by the status code.
	Map() map[string]StatusCode
}

// NodeScore contains which Node got how much score during the scheduling cycle.
type NodeScore interface {
	// Map returns a map
	// which is keyed by the node name and valued by the score.
	Map() map[string]int
}
