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

// Package prefilterextensions exports an api.PreFilterExtensions to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.

package prefilterextensions

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// prefilterextensions is the current plugin assigned with SetPlugin.
var prefilterextensions api.PreFilterExtensions

// SetPlugin is exposed to prevent package cycles.
func SetPlugin(preFilterExtensions api.PreFilterExtensions) {
	if preFilterExtensions == nil {
		panic("nil preFilterExtensions")
	}

	plugin.MustSet(preFilterExtensions)
}

// prevent unused lint errors (lint is run with normal go).
var (
	_ func() uint32 = _addpod
	_ func() uint32 = _removepod
)

// _addPod is only exported to the host.
//
// export addpod
func _addpod() uint32 { //nolint
	if prefilterextensions == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export reserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	status := prefilterextensions.AddPod(cyclestate.Values, cyclestate.Pod, cyclestate.Pod, &filter.NodeInfo{})

	return imports.StatusToCode(status)
}

// _removePod is only exported to the host.
//
// export removepod
func _removepod() uint32 { //nolint
	if prefilterextensions == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export unreserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	status := prefilterextensions.RemovePod(cyclestate.Values, cyclestate.Pod, cyclestate.Pod, &filter.NodeInfo{})

	return imports.StatusToCode(status)
}
