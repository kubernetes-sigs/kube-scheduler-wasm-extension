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
	"unsafe"

	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

func main() {
	plugin := cyclestate{}
	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
	score.SetPlugin(plugin)
}

const (
	// name is the name of the plugin used in the plugin registry and configurations.
	name              = "CycleState"
	prefilterStateKey = "PreFilter" + name
)

// cyclestate makes sure api.CycleState is consistent between callbacks.
type cyclestate struct{}

// podSpec is used to test cycle state coherency.
var podSpec *protoapi.PodSpec

type prefilterStateVal map[string]any

func (cyclestate) PreFilter(state api.CycleState, pod api.Pod) (nodeNames []string, status *api.Status) {
	if nextPodSpec := pod.Spec(); unsafe.Pointer(nextPodSpec) == unsafe.Pointer(podSpec) {
		panic("didn't reset pod on pre-filter")
	} else {
		podSpec = nextPodSpec
	}
	if _, ok := state.Read(prefilterStateKey); ok {
		panic("didn't reset state on pre-filter")
	} else {
		state.Write(prefilterStateKey, prefilterStateVal{})
	}
	return
}

func (cyclestate) Filter(state api.CycleState, pod api.Pod, _ api.NodeInfo) (status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	if val, ok := state.Read(prefilterStateKey); !ok {
		panic("didn't propagate state from pre-filter")
	} else {
		val.(prefilterStateVal)["a"] = struct{}{}
	}
	return
}

func (cyclestate) Score(state api.CycleState, pod api.Pod, _ string) (score int32, status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	if val, ok := state.Read(prefilterStateKey); !ok {
		panic("didn't propagate state from filter")
	} else if _, ok := val.(prefilterStateVal)["a"]; !ok {
		panic("value lost propagating from filter")
	}
	return
}
