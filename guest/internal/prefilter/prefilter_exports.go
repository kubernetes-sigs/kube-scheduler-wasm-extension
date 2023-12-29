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

// Package prefilter is defined internally so that it can export Pod as
// cyclestate.Pod, without circular dependencies or exporting it publicly.
package prefilter

import (
	"runtime"
	"unsafe"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
)

// prefilter is the current plugin assigned with SetPlugin.
var prefilter api.PreFilterPlugin

// SetPlugin is exposed to prevent package cycles.
func SetPlugin(pluginInitializer func(klog klogapi.Klog, jsonConfig []byte, h handleapi.Handle) api.PreFilterPlugin, klog klogapi.Klog, jsonConfig []byte) {
	handle := handle.NewFrameworkHandle()
	prefilter = pluginInitializer(klog, jsonConfig, handle)
	if prefilter == nil {
		panic("nil prefilterPlugin")
	}
	plugin.MustSet(prefilter)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _prefilter

// _prefilter is only exported to the host.
//
//export prefilter
func _prefilter() uint32 { //nolint
	// This function begins a new scheduling cycle: zero out any cycle state.
	currentPod = nil
	currentCycleState = map[string]any{}

	if prefilter == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export prefilter so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	nodeNames, status := prefilter.PreFilter(CycleState, Pod)

	// If plugin returned nodeNames, concatenate them into a C-string and call
	// the host with the count and memory region.
	cString := toNULTerminated(nodeNames)
	if cString != nil {
		ptr := uint32(uintptr(unsafe.Pointer(&cString[0])))
		size := uint32(len(cString))
		setNodeNamesResult(ptr, size)
		runtime.KeepAlive(cString) // until ptr is no longer needed.
	}

	return imports.StatusToCode(status)
}
