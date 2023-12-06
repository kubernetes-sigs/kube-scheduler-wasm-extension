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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/bind"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postbind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prebind"
)

type extensionPoints interface {
	api.PreBindPlugin
	api.BindPlugin
	api.PostBindPlugin
}

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	var plugin extensionPoints = noopPlugin{}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "preBind":
			plugin = preBindPlugin{}
		case "bind":
			plugin = bindPlugin{}
		case "postBind":
			plugin = postBindPlugin{}
		}
	}
	prebind.SetPlugin(func(h handleapi.Handle) api.PreBindPlugin { return plugin })
	bind.SetPlugin(func(h handleapi.Handle) api.BindPlugin { return plugin })
	postbind.SetPlugin(func(h handleapi.Handle) api.PostBindPlugin { return plugin })
}

// noopPlugin doesn't do anything, except evaluate each parameter.
type noopPlugin struct{}

func (noopPlugin) PreBind(state api.CycleState, pod proto.Pod, nodeName string) *api.Status {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeName
	return nil
}

func (noopPlugin) Bind(state api.CycleState, pod proto.Pod, nodeName string) *api.Status {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeName
	return nil
}

func (noopPlugin) PostBind(state api.CycleState, pod proto.Pod, nodeName string) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeName
}

// preBindPlugin returns the length of nodeName
type preBindPlugin struct{ noopPlugin }

func (preBindPlugin) PreBind(_ api.CycleState, _ proto.Pod, nodeName string) *api.Status {
	status := 0
	if nodeName == "bad" {
		status = 1
	}
	return &api.Status{Code: api.StatusCode(status), Reason: "name is " + nodeName}
}

// bindPlugin returns the length of nodeName
type bindPlugin struct{ noopPlugin }

func (bindPlugin) Bind(_ api.CycleState, _ proto.Pod, nodeName string) *api.Status {
	status := 0
	if nodeName == "bad" {
		status = 1
	}
	return &api.Status{Code: api.StatusCode(status), Reason: "name is " + nodeName}
}

// postBindPlugin returns nothing
type postBindPlugin struct{ noopPlugin }

func (postBindPlugin) PostBind(_ api.CycleState, _ proto.Pod, nodeName string) {
	if nodeName == "bad" {
		panic("name is bad")
	}
}
