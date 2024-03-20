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

// Package prefilter exports an api.PreFilterPlugin to the host.
package prefilter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	internalprefilter "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/prefilter"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// SetPlugin should be called in `main` to assign an api.PreFilterPlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := filterPlugin{}
//		prefilter.SetPlugin(plugin)
//		filter.SetPlugin(plugin)
//	}
//
//	type filterPlugin struct{}
//
//	func (filterPlugin) PreFilter(state api.CycleState, pod proto.Pod) (nodeNames []string, status *Status) {
//		// Write state you need on Filter
//	}
//
//	func (filterPlugin) Filter(state api.CycleState, pod api.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
//		var Filter int32
//		// Derive Filter for the node name using state set on PreFilter!
//		return Filter, nil
//	}
//
// Note: This may be set without filter.SetPlugin, if the pre-filter plugin has
// the only filtering logic, or only used to configure api.CycleState.
func SetPlugin(prefilterPlugin api.PreFilterPlugin) {
	internalprefilter.SetPlugin(prefilterPlugin)
}

type podInfoToAdd struct {
	pod proto.Pod
}

func (p *podInfoToAdd) Pod() proto.Pod {
	return p.lazyPodMap()
}

func (p *podInfoToAdd) lazyPodMap() proto.Pod {
	if pod := p.pod; pod != nil {
		return pod
	}

	var msg protoapi.Pod
	if err := imports.Pod(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	p.pod = internalprefilter.Pod
	return p.pod
}
