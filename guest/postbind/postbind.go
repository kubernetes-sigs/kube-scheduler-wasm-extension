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

// Package postbind exports an api.PostBindPlugin to the host.
package postbind

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// postbind is the current plugin assigned with SetPlugin.
var postbind api.PostBindPlugin

// SetPlugin should be called in `main` to assign an api.PostBindPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := bindPlugin{}
//		bind.SetPlugin(func(h handleapi.Handle) api.BindPlugin { return plugin })
//		postbind.SetPlugin(func(h handleapi.Handle) api.PostBindPlugin { return plugin })
//	}
//
//	type bindPlugin struct{}
//
//	func (bindPlugin) Bind(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
//		// Write state you need on Bind
//	}
//
//	func (bindPlugin) PostBind(state api.CycleState, pod proto.Pod, nodeName string) {
//		// Write state you need on PostBind
//	}
func SetPlugin(pluginInitializer func(h handleapi.Handle) api.PostBindPlugin) {
	handle := handle.NewFrameworkHandle()
	postbind = pluginInitializer(handle)
	if postbind == nil {
		panic("nil postbindPlugin")
	}
	plugin.MustSet(postbind)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() = _postbind

// _postbind is only exported to the host.
//
//export postbind
func _postbind() { //nolint
	if postbind == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement PostBindPlugin: return success.
		return
	}

	nodeName := imports.NodeName()
	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	postbind.PostBind(cyclestate.Values, cyclestate.Pod, nodeName)
}
