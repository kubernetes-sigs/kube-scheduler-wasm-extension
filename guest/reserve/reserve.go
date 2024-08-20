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

// Package reserve exports an api.ReservePlugin to the host.
package reserve

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// reserve is the current plugin assigned with SetPlugin.
var reserve api.ReservePlugin

// SetPlugin should be called in `main` to assign an api.ReservePlugin instance.
//
// For example:
//
//	func main() {
//		plugin := reservePlugin{}
//		reserve.SetPlugin(plugin)
//	}
//
//	type reservePlugin struct{}
//
//	func (reservePlugin) Reserve(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
//		// Write state you need on Reserve
//	}
//
//	func (reservePlugin) Unreserve(state api.CycleState, pod proto.Pod, nodeName string) {
//		// Write state you need on Unreserve
//	}
func SetPlugin(reservePlugin api.ReservePlugin) {
	if reservePlugin == nil {
		panic("nil reservePlugin")
	}
	reserve = reservePlugin
	plugin.MustSet(reserve)
}

// prevent unused lint errors (lint is run with normal go).
var (
	_ func() uint32 = _reserve
	_ func()        = _unreserve
)

// _reserve is only exported to the host.
//
//export reserve
func _reserve() uint32 { //nolint
	if reserve == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export reserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	nodeName := imports.CurrentNodeName()
	status := reserve.Reserve(cyclestate.Values, cyclestate.Pod, nodeName)

	return imports.StatusToCode(status)
}

// _unreserve is only exported to the host.
//
//export unreserve
func _unreserve() { //nolint
	if reserve == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export unreserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return
	}

	nodeName := imports.CurrentNodeName()
	reserve.Unreserve(cyclestate.Values, cyclestate.Pod, nodeName)
}
