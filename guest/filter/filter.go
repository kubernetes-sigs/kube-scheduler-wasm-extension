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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// filter is the current plugin assigned with SetPlugin.
var filter api.FilterPlugin

// SetPlugin should be called in `main` to assign an api.FilterPlugin instance.
//
// For example:
//
//	func main() {
//		filter.SetPlugin(nameEqualsPodSpec{})
//	}
//
//	type nameEqualsPodSpec struct{}
//
//	func (nameEqualsPodSpec) Filter(state api.CycleState, pod api.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
//		panic("implement me")
//	}
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
	if filter == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("filter imported, but filter.SetPlugin not called")
	}

	s := filter.Filter(cyclestate.Values, cyclestate.Pod, &nodeInfo{})

	return imports.StatusToCode(s)
}

var _ api.NodeInfo = (*nodeInfo)(nil)

// nodeInfo is lazy so that a plugin which doesn't read fields avoids a
// relatively expensive unmarshal penalty.
//
// Note: Unlike proto.Pod, this is not special cased for the scheduling cycle.
type nodeInfo struct {
	node *protoapi.Node
}

func (n *nodeInfo) Node() proto.Node {
	return n
}

func (n *nodeInfo) Metadata() *meta.ObjectMeta {
	return n.lazyNode().Metadata
}

func (n *nodeInfo) Spec() *protoapi.NodeSpec {
	return n.lazyNode().Spec
}

func (n *nodeInfo) Status() *protoapi.NodeStatus {
	return n.lazyNode().Status
}

// lazyNode lazy initializes node from imports.NodeInfoNode.
func (n *nodeInfo) lazyNode() *protoapi.Node {
	if node := n.node; node != nil {
		return node
	}

	var msg protoapi.Node
	if err := imports.NodeInfoNode(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	n.node = &msg
	return n.node
}
