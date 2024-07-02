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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// prefilter is the current plugin assigned with SetPlugin.
var prefilter api.PreFilterPlugin

// prefilterextensions is the current plugin assigned with SetPlugin.
var prefilterextensions api.PreFilterExtensions

var Pod proto.Pod

// SetPlugin is exposed to prevent package cycles.
func SetPlugin(prefilterPlugin api.PreFilterPlugin) {
	if prefilterPlugin == nil {
		panic("nil prefilterPlugin")
	}
	prefilter = prefilterPlugin
	plugin.MustSet(prefilterPlugin)
}

// prevent unused lint errors (lint is run with normal go).
var (
	_ func() uint32 = _addpod
	_ func() uint32 = _removepod
	_ func() uint32 = _prefilter
)

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

// _addPod is only exported to the host.
//
//export addpod
func _addpod() uint32 { //nolint
	if prefilterextensions == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export reserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	status := prefilterextensions.AddPod(CycleState, Pod, &PodInfo{}, &NodeInfo{})

	return imports.StatusToCode(status)
}

// _removePod is only exported to the host.
//
//export removepod
func _removepod() uint32 { //nolint
	if prefilterextensions == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export unreserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	status := prefilterextensions.RemovePod(CycleState, Pod, &PodInfo{}, &NodeInfo{})

	return imports.StatusToCode(status)
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
