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

// Package filter exports an api.FilterPlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package filter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// filter is the current plugin assigned with SetPlugin.
var filter api.FilterPlugin

// SetPlugin should be called in `main` to assign an api.FilterPlugin instance.
//
// For example:
//
//	func main() {
//		filter.SetPlugin(filterPlugin{})
//	}
//
//	type filterPlugin struct{}
//
//	func (filterPlugin) Filter(state api.CycleState, pod api.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
//		panic("implement me")
//	}
//
// Note: If you need state, you can assign it with prefilter.SetPlugin.
func SetPlugin(filterPlugin api.FilterPlugin) {
	if filterPlugin == nil {
		panic("nil filterPlugin")
	}
	filter = filterPlugin
	plugin.MustSet(filter)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _filter

// filter is only exported to the host.
//
//export filter
func _filter() uint32 { //nolint
	if filter == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement FilterPlugin: return success.
		return 0
	}

	nodename := imports.CurrentNodeName()
	if nodename == "" {
		return imports.StatusToCode(&api.Status{Code: api.StatusCodeError, Reason: "could not get current node name"})
	}

	s := filter.Filter(cyclestate.Values, cyclestate.Pod, sharedlister.NodeInfos().Get(nodename))

	return imports.StatusToCode(s)
}
