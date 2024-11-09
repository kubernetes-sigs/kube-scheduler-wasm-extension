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

// Package main is the entrypoint of the %.wasm file, compiled with
// '-target=wasi'. See /guest/RATIONALE.md for details.
package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/helper"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/util"
	k8sproto "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

const (
	// ErrReasonUnknownCondition is used for NodeUnknownCondition predicate error.
	ErrReasonUnknownCondition = "node(s) had unknown conditions"
	// ErrReasonUnschedulable is used for NodeUnschedulable predicate error.
	ErrReasonUnschedulable = "node(s) were unschedulable"
)

func main() {
	p, err := New(klog.Get(), config.Get())
	if err != nil {
		panic(err)
	}
	plugin.Set(p)
}

func New(klog klogapi.Klog, jsonConfig []byte) (api.Plugin, error) {
	return &NodeUnschedulable{}, nil
}

// NodeUnschedulable plugin filters nodes that set node.Spec.Unschedulable=true unless
// the pod tolerates {key=node.kubernetes.io/unschedulable, effect:NoSchedule} taint.
type NodeUnschedulable struct{}

func (p *NodeUnschedulable) Name() string {
	return "NodeUnschedulable"
}

func (p *NodeUnschedulable) Filter(_ api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) *api.Status {
	node := nodeInfo.Node()
	if node == nil {
		return &api.Status{
			Code:   api.StatusCodeUnschedulableAndUnresolvable,
			Reason: ErrReasonUnknownCondition,
		}
	}
	// If pod tolerate unschedulable taint, it's also tolerate `node.Spec.Unschedulable`.
	podToleratesUnschedulable := helper.TolerationsTolerateTaint(pod.Spec().Tolerations, &k8sproto.Taint{
		Key:    util.To(api.TaintNodeUnschedulable),
		Effect: util.To(api.TaintEffectNoSchedule),
	})
	unschedulable := false
	if node.Spec().Unschedulable != nil {
		unschedulable = *node.Spec().Unschedulable
	}
	if unschedulable && !podToleratesUnschedulable {
		return &api.Status{
			Code:   api.StatusCodeUnschedulableAndUnresolvable,
			Reason: ErrReasonUnschedulable,
		}
	}
	return nil
}
