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

// Package prefilterextensions exports an api.PreFilterExtensions to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.

package prefilterextensions

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// prefilterextensions is the current plugin assigned with SetPlugin.
var prefilterextensions api.PreFilterExtensions

// SetPlugin is exposed to prevent package cycles.
func SetPlugin(preFilterExtensions api.PreFilterExtensions) {
	if preFilterExtensions == nil {
		panic("nil preFilterExtensions")
	}
	prefilterextensions = preFilterExtensions
	plugin.MustSet(prefilterextensions)
}

// prevent unused lint errors (lint is run with normal go).
var (
	_ func() uint32 = _addpod
	_ func() uint32 = _removepod
)

// _addPod is only exported to the host.
//
//export addpod
func _addpod() uint32 { //nolint
	if prefilterextensions == nil { // Then, the user didn't define one.
		// Unlike most plugins we always export reserve so that we can reset
		// the cycle state: return success to avoid no-op overhead.
		return 0
	}

	nodename := imports.CurrentNodeName()
	if nodename == "" {
		return imports.StatusToCode(&api.Status{Code: api.StatusCodeError, Reason: "could not get current node name"})
	}

	status := prefilterextensions.AddPod(cyclestate.Values, cyclestate.Pod, &podInfo{}, sharedlister.NodeInfos().Get(nodename))

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

	nodename := imports.CurrentNodeName()
	if nodename == "" {
		return imports.StatusToCode(&api.Status{Code: api.StatusCodeError, Reason: "could not get current node name"})
	}

	status := prefilterextensions.RemovePod(cyclestate.Values, cyclestate.Pod, &podInfo{}, sharedlister.NodeInfos().Get(nodename))

	return imports.StatusToCode(status)
}

type podInfo struct {
	pod proto.Pod
}

func (p *podInfo) GetApiVersion() string {
	return p.lazyPod().GetApiVersion()
}

func (p *podInfo) GetKind() string {
	return p.lazyPod().GetKind()
}

func (p *podInfo) GetName() string {
	return p.lazyPod().GetName()
}

func (p *podInfo) GetNamespace() string {
	return p.lazyPod().GetNamespace()
}

func (p *podInfo) GetResourceVersion() string {
	return p.lazyPod().GetNamespace()
}

func (p *podInfo) GetUid() string {
	return p.lazyPod().GetUid()
}

func (p *podInfo) Pod() proto.Pod {
	return p.lazyPod()
}

func (p *podInfo) Spec() *protoapi.PodSpec {
	return p.lazyPod().Spec()
}

func (p *podInfo) Status() *protoapi.PodStatus {
	return p.lazyPod().Status()
}

// lazyPod lazy initializes pod from imports.Pod.
func (p *podInfo) lazyPod() proto.Pod {
	if pod := p.pod; pod != nil {
		return pod
	}

	var msg protoapi.Pod
	if err := imports.TargetPod(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	p.pod = &internalproto.Pod{Msg: &msg}
	return p.pod
}
