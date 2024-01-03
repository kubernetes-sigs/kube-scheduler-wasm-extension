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

// Package prebind exports an api.PreBindPlugin to the host.
package prebind

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
)

// prebind is the current plugin assigned with SetPlugin.
var prebind api.PreBindPlugin

// SetPlugin should be called in `main` to assign an api.PreBindPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := bindPlugin{}
//		bind.SetPlugin(func(klog klogapi.Klog, jsonConfig []byte, h handleapi.Handle) (api.Plugin, error) { return plugin, nil })
//		prebind.SetPlugin(func(klog klogapi.Klog, jsonConfig []byte, h handleapi.Handle) (api.Plugin, error) { return plugin, nil })
//	}
//
//	type bindPlugin struct{}
//
//	func (bindPlugin) Bind(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
//		// Write state you need on Bind
//	}
//
//	func (bindPlugin) PreBind(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
//		// Write state you need on Bind
//	}
func SetPlugin(pluginFactory handleapi.PluginFactory) {
	handle := handle.NewFrameworkHandle()
	p, err := pluginFactory(klog.Get(), config.Get(), handle)
	if err != nil {
		panic(err)
	}
	var ok bool
	prebind, ok = p.(api.PreBindPlugin)
	if !ok || prebind == nil {
		panic("nil PreBindPlugin or a plugin is not compatible with PreBindPlugin type")
	}
	plugin.MustSet(prebind)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _prebind

// _prebind is only exported to the host.
//
//export prebind
func _prebind() uint32 { //nolint
	if prebind == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement PreBindPlugin: return success.
		return 0
	}

	nodeName := imports.NodeName()
	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	s := prebind.PreBind(cyclestate.Values, cyclestate.Pod, nodeName)

	return imports.StatusToCode(s)
}
