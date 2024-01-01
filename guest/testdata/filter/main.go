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

package main

import (
	"os"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postfilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilterextensions"
)

type extensionPoints interface {
	api.PreFilterPlugin
	api.FilterPlugin
	api.PostFilterPlugin
	api.PreFilterExtensions
}

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	var plugin extensionPoints = noopPlugin{}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "filter":
			plugin = filterPlugin{}
		case "preFilter":
			plugin = preFilterPlugin{}
		case "postFilter":
			plugin = postFilterPlugin{}
		case "preFilterExtensions":
			plugin = preFilterExtensions{}
		}
	}
	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
	postfilter.SetPlugin(plugin)
	prefilterextensions.SetPlugin(plugin)
}

// noopPlugin doesn't do anything, except evaluate each parameter.
type noopPlugin struct{}

func (noopPlugin) PreFilter(state api.CycleState, pod proto.Pod) (nodeNames []string, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	return
}

func (noopPlugin) Filter(state api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeInfo.Node().Spec() // trigger lazy loading
	return
}

func (noopPlugin) PostFilter(state api.CycleState, pod proto.Pod, nodeMap api.NodeToStatus) (nominatedNodeName string, nominatingMode api.NominatingMode, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeMap.Map()
	return
}

func (noopPlugin) AddPod(state api.CycleState, pod proto.Pod, podInfoToAdd proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeInfo.Node().Spec() // trigger lazy loading
	return
}

func (noopPlugin) RemovePod(state api.CycleState, pod proto.Pod, podInfoToAdd proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeInfo.Node().Spec() // trigger lazy loading
	return
}

// preFilterPlugin schedules a node if its name equals its pod spec.
type preFilterPlugin struct{ noopPlugin }

func (preFilterPlugin) PreFilter(_ api.CycleState, pod proto.Pod) ([]string, *api.Status) {
	// First, check if the pod spec node name is empty. If so, pass!
	podSpecNodeName := pod.Spec().GetNodeName()
	if len(podSpecNodeName) == 0 {
		return nil, nil
	}
	return []string{podSpecNodeName}, nil
}

// filterPlugin schedules a node if its name equals its pod spec.
type filterPlugin struct{ noopPlugin }

func (filterPlugin) Filter(_ api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) *api.Status {
	// First, check if the pod spec node name is empty. If so, pass!
	podSpecNodeName := pod.Spec().GetNodeName()
	if len(podSpecNodeName) == 0 {
		return nil
	}

	// Next, check if the node name matches the spec node. If so, pass!
	nodeName := nodeInfo.Node().GetName()
	if podSpecNodeName == nodeName {
		return nil
	}

	// Otherwise, this is unschedulable, so note the reason.
	return &api.Status{
		Code:   api.StatusCodeUnschedulable,
		Reason: podSpecNodeName + " != " + nodeName,
	}
}

type postFilterPlugin struct{ noopPlugin }

func (postFilterPlugin) PostFilter(_ api.CycleState, pod proto.Pod, nodeMap api.NodeToStatus) (string, api.NominatingMode, *api.Status) {
	// First, check if the pod spec node name is empty. If so, pass!
	podSpecNodeName := pod.Spec().GetNodeName()
	if len(podSpecNodeName) == 0 {
		return "", 0, nil
	}
	m := nodeMap.Map()
	if m == nil {
		return "", 0, nil
	}
	// If nominatedNodeName is schedulable, pass!
	if val, ok := m[podSpecNodeName]; ok {
		if val == 0 {
			return podSpecNodeName, api.ModeOverride, nil
		}
	}
	// Otherwise, this is unschedulableAndUnresolvable, so note the reason.
	return podSpecNodeName, api.ModeNoop, &api.Status{
		Code:   api.StatusCodeUnschedulableAndUnresolvable,
		Reason: podSpecNodeName + " is unschedulable",
	}
}

type preFilterExtensions struct{ noopPlugin }

func (preFilterExtensions) AddPod(state api.CycleState, pod proto.Pod, podInfoToAdd proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = podInfoToAdd
	_ = nodeInfo
	return
}

func (preFilterExtensions) RemovePod(state api.CycleState, pod proto.Pod, podInfoToAdd proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = podInfoToAdd
	_ = nodeInfo
	return
}
