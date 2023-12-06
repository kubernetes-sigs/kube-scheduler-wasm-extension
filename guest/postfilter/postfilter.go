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

// Package postfilter exports an api.PostFilterPlugin to the host.
package postfilter

import (
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
)

// postfilter is the current plugin assigned with SetPlugin.
var postfilter api.PostFilterPlugin

// SetPlugin should be called in `main` to assign an api.PostFilterPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := filterPlugin{}
//		postfilter.SetPlugin(func(h handleapi.Handle) api.PostFilterPlugin { return plugin })
//		filter.SetPlugin(func(h handleapi.Handle) api.FilterPlugin { return plugin })
//	}
//
//	type filterPlugin struct{}
//
//	func (filterPlugin) PostFilter(state api.CycleState, pod proto.Pod, filteredNodeStatusMap api.NodeToStatus) (int32, status *api.Status) {
//		// Write state you need on Filter
//	}
//
//	func (filterPlugin) Filter(state api.CycleState, pod api.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
//		var Filter int32
//		// Derive Filter for the node name using state set on PreFilter!
//		return Filter, nil
//	}
func SetPlugin(pluginInitializer func(h handleapi.Handle) api.PostFilterPlugin) {
	handle := handle.NewFrameworkHandle()
	postfilter = pluginInitializer(handle)
	if postfilter == nil {
		panic("nil postfilterPlugin")
	}
	plugin.MustSet(postfilter)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint64 = _postfilter

// _postfilter is only exported to the host.
//
//export postfilter
func _postfilter() uint64 { //nolint

	if postfilter == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export postfilter so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	nominatedNodeName, nominatingMode, status := postfilter.PostFilter(cyclestate.Values, cyclestate.Pod, &nodeToStatus{})
	ptr, size := mem.StringToPtr(nominatedNodeName)
	setNominatedNodeNameResult(ptr, size)
	runtime.KeepAlive(nominatedNodeName) // until ptr is no longer needed.

	return (uint64(nominatingMode) << uint64(32)) | uint64(imports.StatusToCode(status))
}

type nodeToStatus struct {
	statusMap map[string]api.StatusCode
}

func (n *nodeToStatus) Map() map[string]api.StatusCode {
	return n.lazyNodeToStatusMap()
}

// lazyNodeToStatusMap returns NodeToStatusMap from imports.NodeToStatusMap.
func (n *nodeToStatus) lazyNodeToStatusMap() map[string]api.StatusCode {
	nodeMap := imports.NodeToStatusMap()
	n.statusMap = nodeMap
	return n.statusMap
}
