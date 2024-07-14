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
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
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

	s := filter.Filter(cyclestate.Values, cyclestate.Pod, &NodeInfo{})

	return imports.StatusToCode(s)
}

var _ api.NodeInfo = (*NodeInfo)(nil)

// nodeInfo is lazy so that a plugin which doesn't read fields avoids a
// relatively expensive unmarshal penalty.
//
// Note: Unlike proto.Pod, this is not special cased for the scheduling cycle.
type NodeInfo struct {
	node proto.Node
}

func (n *NodeInfo) GetUid() string {
	return n.lazyNode().GetUid()
}

func (n *NodeInfo) GetName() string {
	return n.lazyNode().GetName()
}

func (n *NodeInfo) GetNamespace() string {
	return n.lazyNode().GetNamespace()
}

func (n *NodeInfo) GetResourceVersion() string {
	return n.lazyNode().GetResourceVersion()
}

func (n *NodeInfo) Node() proto.Node {
	return n.lazyNode()
}

// lazyNode lazy initializes node from imports.Node.
func (n *NodeInfo) lazyNode() proto.Node {
	if node := n.node; node != nil {
		return node
	}

	var msg protoapi.Node
	if err := imports.Node(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	n.node = &internalproto.Node{Msg: &msg}
	return n.node
}

type PodInfo struct {
	pod proto.Pod
}

func (p *PodInfo) GetApiVersion() string {
	return p.lazyPod().GetApiVersion()
}

func (p *PodInfo) GetKind() string {
	return p.lazyPod().GetKind()
}

func (p *PodInfo) GetName() string {
	return p.lazyPod().GetName()
}

func (p *PodInfo) GetNamespace() string {
	return p.lazyPod().GetNamespace()
}

func (p *PodInfo) GetResourceVersion() string {
	return p.lazyPod().GetNamespace()
}

func (p *PodInfo) GetUid() string {
	return p.lazyPod().GetUid()
}

func (p *PodInfo) Pod() proto.Pod {
	return p.lazyPod()
}

func (p *PodInfo) Spec() *protoapi.PodSpec {
	return p.lazyPod().Spec()
}

func (p *PodInfo) Status() *protoapi.PodStatus {
	return p.lazyPod().Status()
}

// lazyPod lazy initializes pod from imports.Pod.
func (p *PodInfo) lazyPod() proto.Pod {
	if pod := p.pod; pod != nil {
		return pod
	}

	var msg protoapi.Pod
	if err := imports.Pod(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	p.pod = &internalproto.Pod{Msg: &msg}
	return p.pod
}
