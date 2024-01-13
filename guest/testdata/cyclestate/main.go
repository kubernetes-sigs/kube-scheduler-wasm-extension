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

// Package main tests state propagation through defined extension points.
// See https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#extension-points
package main

import (
	"os"
	"time"
	"unsafe"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/bind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/permit"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postbind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postfilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prebind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/reserve"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/scoreextensions"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "0":
		case "1":
			clusterEvents = []api.ClusterEvent{
				{Resource: api.PersistentVolume, ActionType: api.Delete},
			}
		case "2":
			clusterEvents = []api.ClusterEvent{
				{Resource: api.Node, ActionType: api.Add},
				{Resource: api.PersistentVolume, ActionType: api.Delete},
			}
		default:
			panic("unsupported count")
		}
	}

	plugin := statePlugin{}
	enqueue.SetPlugin(plugin)
	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
	postfilter.SetPlugin(plugin)
	prescore.SetPlugin(plugin)
	score.SetPlugin(plugin)
	scoreextensions.SetPlugin(plugin)
	reserve.SetPlugin(plugin)
	permit.SetPlugin(plugin)
	prebind.SetPlugin(plugin)
	bind.SetPlugin(plugin)
	postbind.SetPlugin(plugin)
}

const (
	// name is the name of the plugin used in the plugin registry and configurations.
	name              = "CycleState"
	preFilterStateKey = "PreFilter" + name
	preScoreStateKey  = "PreScore" + name
	preBindStateKey   = "PreBind" + name
)

// statePlugin makes sure api.CycleState is consistent between callbacks.
type statePlugin struct{}

var clusterEvents []api.ClusterEvent

func (statePlugin) EventsToRegister() []api.ClusterEvent { return clusterEvents }

// podSpec is used to test cycle state coherency.
var podSpec *protoapi.PodSpec

type preFilterStateVal map[string]any

type preScoreStateVal map[string]any

type preBindStateVal map[string]any

func (statePlugin) PreFilter(state api.CycleState, pod proto.Pod) (nodeNames []string, status *api.Status) {
	nextPodSpec := pod.Spec()
	if unsafe.Pointer(nextPodSpec) == unsafe.Pointer(podSpec) {
		panic("didn't reset pod on pre-filter")
	}
	podSpec = nextPodSpec
	mustNotScoreState(state)
	if _, ok := state.Read(preFilterStateKey); ok {
		panic("didn't reset filter state on pre-filter")
	} else {
		state.Write(preFilterStateKey, preFilterStateVal{})
	}
	return
}

func (statePlugin) Filter(state api.CycleState, pod proto.Pod, _ api.NodeInfo) (status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	mustNotScoreState(state)
	if val, ok := state.Read(preFilterStateKey); !ok {
		panic("didn't propagate filter state from pre-filter")
	} else {
		val.(preFilterStateVal)["filter"] = struct{}{}
	}
	return
}

func (statePlugin) PostFilter(state api.CycleState, pod proto.Pod, _ api.NodeToStatus) (nominatedNodeName string, nominatingMode api.NominatingMode, status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	mustNotScoreState(state)
	if val, ok := state.Read(preFilterStateKey); !ok {
		panic("didn't propagate state from pre-filter")
	} else if _, ok = val.(preFilterStateVal)["filter"]; !ok {
		panic("filter value lost propagating from filter")
	}
	return
}

func (statePlugin) PreScore(state api.CycleState, pod proto.Pod, _ proto.NodeList) *api.Status {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	mustFilterState(state)
	if _, ok := state.Read(preScoreStateKey); ok {
		panic("didn't reset score state on pre-score")
	} else {
		state.Write(preScoreStateKey, preScoreStateVal{})
	}
	return nil
}

func (statePlugin) Score(state api.CycleState, pod proto.Pod, _ string) (score int32, status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	mustFilterState(state)
	if val, ok := state.Read(preScoreStateKey); !ok {
		panic("didn't propagate score state from pre-score")
	} else {
		val.(preScoreStateVal)["score"] = struct{}{}
	}
	return
}

func (statePlugin) NormalizeScore(state api.CycleState, pod proto.Pod, _ api.NodeScore) (scores map[string]int, status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	mustFilterState(state)
	if val, ok := state.Read(preScoreStateKey); !ok {
		panic("didn't propagate score state from pre-score")
	} else if _, ok = val.(preScoreStateVal)["score"]; !ok {
		panic("score value lost propagating from score-extensions")
	} else {
		val.(preScoreStateVal)["scoreextensions"] = struct{}{}
	}
	return
}

func (statePlugin) Reserve(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status) {
	// Actually, it is not called from the cycle test, but it needs for Reserve plugin.
	return
}

func (statePlugin) Unreserve(state api.CycleState, pod proto.Pod, nodeName string) {
	mustFilterState(state)
}
func (statePlugin) Permit(state api.CycleState, pod proto.Pod, nodeName string) (status *api.Status, timeout time.Duration) {
	// Actually, it is not called from the cycle test, but it needs for Permit plugin.
	return
}

func (statePlugin) PreBind(state api.CycleState, pod proto.Pod, _ string) (status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	if _, ok := state.Read(preBindStateKey); ok {
		panic("didn't reset pre-bind state on pre-bind")
	} else {
		state.Write(preBindStateKey, preBindStateVal{})
	}
	return
}

func (statePlugin) Bind(state api.CycleState, pod proto.Pod, _ string) (status *api.Status) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	if val, ok := state.Read(preBindStateKey); !ok {
		panic("didn't propagate pre-bind state from pre-bind")
	} else {
		val.(preBindStateVal)["bind"] = struct{}{}
	}
	return
}

func (statePlugin) PostBind(state api.CycleState, pod proto.Pod, _ string) {
	if unsafe.Pointer(pod.Spec()) != unsafe.Pointer(podSpec) {
		panic("didn't cache pod from pre-filter")
	}
	if val, ok := state.Read(preBindStateKey); !ok {
		panic("didn't propagate pre-bind state from pre-bind")
	} else if _, ok = val.(preBindStateVal)["bind"]; !ok {
		panic("bind value lost propagating from bind")
	}
}

// mustNotScoreState ensures that score state, written after filter, cannot
// be read by extension points before it.
//
// Note: Tests will need to be revisited when plugins become re-entrant for
// reasons such as preemption!
func mustNotScoreState(state api.CycleState) {
	if _, ok := state.Read(preScoreStateKey); ok {
		panic("didn't reset score state on pre-filter")
	}
}

// mustFilterState ensures that score, which happens after filter, can still
// see state written before it.
func mustFilterState(state api.CycleState) {
	if val, ok := state.Read(preFilterStateKey); !ok {
		panic("didn't propagate state from pre-filter")
	} else if _, ok = val.(preFilterStateVal)["filter"]; !ok {
		panic("filter value lost propagating from pre-filter")
	}
}
