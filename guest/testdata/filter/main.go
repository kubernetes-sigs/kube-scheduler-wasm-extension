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

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	"os"

	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
)

type filterPlugin interface {
	api.PreFilterPlugin
	api.FilterPlugin
}

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	var plugin filterPlugin = noop{}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "nameEqualsPodSpec":
			plugin = nameEqualsPodSpec{}
		case "podSpecName":
			plugin = podSpecName{}
		}
	}
	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
}

// noop doesn't do anything, except evaluate each parameter.
type noop struct{}

func (noop) PreFilter(state api.CycleState, pod proto.Pod) (nodeNames []string, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	return
}

func (noop) Filter(state api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeInfo.Node().Spec() // trigger lazy loading
	return
}

// podSpecName schedules a node if its name equals its pod spec.
type podSpecName struct{ noop }

func (podSpecName) PreFilter(_ api.CycleState, pod proto.Pod) ([]string, *api.Status) {
	// First, check if the pod spec node name is empty. If so, pass!
	podSpecNodeName := nilToEmpty(pod.Spec().NodeName)
	if len(podSpecNodeName) == 0 {
		return nil, nil
	}
	return []string{podSpecNodeName}, nil
}

// nameEqualsPodSpec schedules a node if its name equals its pod spec.
type nameEqualsPodSpec struct{ noop }

func (nameEqualsPodSpec) Filter(_ api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) *api.Status {
	// First, check if the pod spec node name is empty. If so, pass!
	podSpecNodeName := nilToEmpty(pod.Spec().NodeName)
	if len(podSpecNodeName) == 0 {
		return nil
	}

	// Next, check if the node name matches the spec node. If so, pass!
	nodeName := nilToEmpty(nodeInfo.Node().Metadata().Name)
	if podSpecNodeName == nodeName {
		return nil
	}

	// Otherwise, this is unschedulable, so note the reason.
	return &api.Status{
		Code:   api.StatusCodeUnschedulable,
		Reason: podSpecNodeName + " != " + nodeName,
	}
}

func nilToEmpty(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}
