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

package score

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
)

// prevent unused lint errors (lint is run with normal go).
var _ func() uint64 = score

var plugin api.ScorePlugin

// score is only exported to the host.
//
//export score
func score() uint64 {
	if plugin == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("score imported, but score.SetPlugin not called")
	}

	// Pod is lazy and the same value for all plugins in a scheduling cycle.
	pod := cyclestate.Pod

	// For ergonomics, we eagerly fetch the nodeName vs making a lazy string.
	// This is less awkward than a lazy string. It is possible in a future
	// refactor we can get this from a `nodeInfo.Node().Metadata.Name` cached
	// in an upstream plugin stage.
	nodeName := imports.NodeName()
	score, status := plugin.Score(pod, nodeName)

	// Pack the score and status code into a single WebAssembly 1.0 compatible
	// result
	return (uint64(score) << uint64(32)) | uint64(imports.StatusToCode(status))
}
