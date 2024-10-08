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

// Package permit exports an api.PermitPlugin to the host.
package permit

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// permit is the current plugin assigned with SetPlugin.
var permit api.PermitPlugin

// SetPlugin should be called in `main` to assign an api.PermitPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := permitPlugin{}
//		permit.SetPlugin(plugin)
//	}
//
//	type permitPlugin struct{}
//
//	func (permitPlugin) Permit(state api.CycleState, p proto.Pod, nodeName string) (status *api.Status, timeout uint32)
//		// Write state you need on Permit
//	}
func SetPlugin(permitPlugin api.PermitPlugin) {
	if permitPlugin == nil {
		panic("nil permitPlugin")
	}
	permit = permitPlugin
	plugin.MustSet(permit)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint64 = _permit

// _permit is only exported to the host.
//
//export permit
func _permit() uint64 {
	if permit == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export permit so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	pod := cyclestate.Pod
	nodeName := imports.CurrentNodeName()
	status, timeout := permit.Permit(cyclestate.Values, pod, nodeName)

	// Pack the score and status code into a single WebAssembly 1.0 compatible
	// result
	return (uint64(imports.StatusToCode(status)) << uint64(32)) | uint64(timeout)
}
