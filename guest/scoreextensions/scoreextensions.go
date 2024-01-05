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

// Package scoreextensions exports an api.ScoreExtensions to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.

package scoreextensions

import (
	"encoding/json"
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// scoreextensions is the current plugin assigned with SetPlugin.
var scoreextensions api.ScoreExtensions

// SetPlugin should be called in `main` to assign an api.ScoreExtensionsPlugin instance.
//
// For example:
//
//	func main() {
//		scoreextensions.SetPlugin(scoreExtensionsPlugin{})
//	}
//
//	type scoreExtensionsPlugin struct{}
//
//	func (scoreExtensionsPlugin) NormalizeScore(state api.CycleState, pod api.Pod, nodeScoreList map[string]int) (map[string]int, *api.Status) {
//		panic("implement me")
//	}
func SetPlugin(scoreExtensions api.ScoreExtensions) {
	if scoreExtensions == nil {
		panic("nil scoreExtensions")
	}
	scoreextensions = scoreExtensions
	plugin.MustSet(scoreextensions)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _normalizescore

// normalizescore is only exported to the host.
//
//export normalizescore
func _normalizescore() uint32 {
	if scoreextensions == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement ScoreExtensionsPlugin: return success and score zero.
		return 0
	}

	// Pod is lazy and the same value for all plugins in a scheduling cycle.
	pod := cyclestate.Pod
	// For ergonomics, we eagerly fetch the nodeScoreList vs making a lazy string.
	// This is less awkward than a lazy string.
	updatedNodeScoreList, status := scoreextensions.NormalizeScore(cyclestate.Values, pod, &nodeScore{})

	jsonByte, err := json.Marshal(updatedNodeScoreList)
	if err != nil {
		panic(err)
	}
	jsonStr := string(jsonByte)
	ptr, size := mem.StringToPtr(jsonStr)
	setNormalizedScoreListResult(ptr, size)
	runtime.KeepAlive(jsonStr) // until ptr is no longer needed.
	// Pack the score and status code into a single WebAssembly 1.0 compatible
	// result
	return imports.StatusToCode(status)
}

type nodeScore struct {
	nodeScoreMap map[string]int
}

func (n *nodeScore) Map() map[string]int {
	return n.lazyNodeScoreList()
}

// lazyNodeScoreList returns NodeScoreList from imports.NodeScoreList.
func (n *nodeScore) lazyNodeScoreList() map[string]int {
	nodeMap := imports.NodeScoreList()
	n.nodeScoreMap = nodeMap
	return n.nodeScoreMap
}
