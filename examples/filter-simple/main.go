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
	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
)

func main() {
	filter.SetPlugin(nameEqualsPodSpec{})
}

// nameEqualsPodSpec schedules a node if its name equals its pod spec.
type nameEqualsPodSpec struct{}

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
