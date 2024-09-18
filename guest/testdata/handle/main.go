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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
)

type extensionPoints interface {
	api.PreFilterPlugin
	api.FilterPlugin
}

// This code is for checking the method of handle.
func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	var plugin extensionPoints = noopPlugin{}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "rejectWaitingPod":
			plugin = pluginForReject{}
		case "getWaitingPod":
			plugin = pluginForGet{}
		}
	}
	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
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

// pluginForReject checks the function of RejectWaitingPod
type pluginForReject struct{ noopPlugin }

func (pluginForReject) Filter(_ api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) *api.Status {
	// Call RejectWaitingPod first
	IsRejected := handle.RejectWaitingPod(pod.GetUid())

	if IsRejected {
		// This is being skipped, note the reason.
		return &api.Status{
			Code:   api.StatusCodeSkip,
			Reason: "UID is " + pod.GetUid(),
		}
	}

	// Otherwise, this is success.
	return &api.Status{
		Code: api.StatusCodeSuccess,
	}
}

// pluginForGet checks the function of GetWaitingPod
type pluginForGet struct{ noopPlugin }

func (pluginForGet) Filter(_ api.CycleState, pod proto.Pod, nodeInfo api.NodeInfo) *api.Status {
	// Call GetWaitingPod first
	waitingPod := handle.GetWaitingPod(pod.GetUid())

	if waitingPod == nil {
		// This is being skipped, note the reason.
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: "UID is " + pod.GetUid(),
		}
	}

	return &api.Status{
		Code: api.StatusCodeSuccess,
	}
}
