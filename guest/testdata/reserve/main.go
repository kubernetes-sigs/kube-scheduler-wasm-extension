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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	klog "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/reserve"
)

type extensionPoints interface {
	api.ReservePlugin
}

func main() {
	var plugin extensionPoints = reservePlugin{}
	reserve.SetPlugin(func(klog klog.Klog, jsonConfig []byte, h handleapi.Handle) api.ReservePlugin { return plugin })
}

type reservePlugin struct{}

func (reservePlugin) Reserve(state api.CycleState, pod proto.Pod, nodeName string) *api.Status {
	state.Write(nodeName+pod.Spec().GetNodeName(), "ok")
	status := 0
	if nodeName == "bad" {
		status = 1
	}
	return &api.Status{Code: api.StatusCode(status), Reason: "name is " + nodeName}
}

func (reservePlugin) Unreserve(state api.CycleState, pod proto.Pod, nodeName string) {
	val, ok := state.Read(nodeName + pod.Spec().GetNodeName())
	if ok && val == "ok" {
		state.Delete(nodeName + pod.Spec().GetNodeName())
		return
	}

	panic("unreserve failed")
}
