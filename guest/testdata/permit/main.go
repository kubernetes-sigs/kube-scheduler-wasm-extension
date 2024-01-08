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
	"time"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/permit"
)

type extensionPoints interface {
	api.PermitPlugin
}

func main() {
	var plugin extensionPoints = permitPlugin{}
	permit.SetPlugin(plugin)
}

type permitPlugin struct{}

func (permitPlugin) Permit(state api.CycleState, pod proto.Pod, nodeName string) (*api.Status, time.Duration) {
	status, timeout := api.StatusCodeSuccess, time.Duration(0)
	if nodeName == "bad" {
		status = api.StatusCodeError
	} else if nodeName == "wait" {
		status = api.StatusCodeWait
		timeout = 10 * time.Second
	}
	return &api.Status{Code: status, Reason: "name is " + nodeName}, timeout
}
