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

package api

import (
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// FilterPlugin is a WebAssembly implementation of framework.FilterPlugin.
type FilterPlugin interface {
	Filter(nodeInfo NodeInfo, pod Pod) *Status
}

// FilterFunc adapts an ordinary function to a FilterPlugin.
type FilterFunc func(nodeInfo NodeInfo, pod Pod) *Status

// Filter returns f(a).
func (f FilterFunc) Filter(nodeInfo NodeInfo, pod Pod) *Status {
	return f(nodeInfo, pod)
}

type NodeInfo interface {
	Node() *protoapi.Node
}

type Pod interface {
	Metadata() *meta.ObjectMeta
	Spec() *protoapi.PodSpec
	Status() *protoapi.PodStatus
}
