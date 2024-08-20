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

// Package bind exports an api.BindPlugin to the host.
package bind

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// bind is the current plugin assigned with SetPlugin.
var bind api.BindPlugin

// SetPlugin should be called in `main` to assign an api.BindPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := bindPlugin{}
//		bind.SetPlugin(plugin)
//	}
//
//	type bindPlugin struct{}
//
//	func (bindPlugin) Bind(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
//		// Write state you need on Bind
//	}
func SetPlugin(bindPlugin api.BindPlugin) {
	if bindPlugin == nil {
		panic("nil bindPlugin")
	}
	bind = bindPlugin
	plugin.MustSet(bind)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _bind

// _bind is only exported to the host.
//
//export bind
func _bind() uint32 { //nolint
	if bind == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement BindPlugin: return success.
		return 0
	}

	nodeName := imports.CurrentNodeName()
	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	s := bind.Bind(cyclestate.Values, cyclestate.Pod, nodeName)

	return imports.StatusToCode(s)
}
