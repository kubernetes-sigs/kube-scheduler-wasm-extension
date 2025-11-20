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

// Package score exports an api.ScorePlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package score

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

const (
	// MaxNodeScore is the maximum score a Score plugin is expected to return.
	MaxNodeScore int64 = 100
)

// score is the current plugin assigned with SetPlugin.
var score api.ScorePlugin

// SetPlugin should be called in `main` to assign an api.ScorePlugin instance.
//
// For example:
//
//	func main() {
//		score.SetPlugin(scorePlugin{})
//	}
//
//	type scorePlugin struct{}
//
//	func (scorePlugin) Score(state api.CycleState, pod api.Pod, nodeName string) (score int32, status *api.Status) {
//		panic("implement me")
//	}
//
// Note: If you need state, you can assign it with prescore.SetPlugin.
func SetPlugin(scorePlugin api.ScorePlugin) {
	if scorePlugin == nil {
		panic("nil scorePlugin")
	}
	score = scorePlugin
	plugin.MustSet(score)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint64 = _score

// score is only exported to the host.
//
//export score
func _score() uint64 {
	if score == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement ScorePlugin: return success and score zero.
		return 0
	}

	// Pod is lazy and the same value for all plugins in a scheduling cycle.
	pod := cyclestate.Pod

	// For ergonomics, we eagerly fetch the nodeName vs making a lazy string.
	// This is less awkward than a lazy string. It is possible in a future
	// refactor we can get this from a `nodeInfo.Node().Metadata.Name` cached
	// in an upstream plugin stage.
	nodeName := imports.CurrentNodeName()
	nodeInfo := sharedlister.NodeInfos().Get(nodeName)
	score, status := score.Score(cyclestate.Values, pod, nodeInfo)

	// Pack the score and status code into a single WebAssembly 1.0 compatible
	// result
	return (uint64(score) << uint64(32)) | uint64(imports.StatusToCode(status))
}
