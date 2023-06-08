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
	Filter(pod Pod, nodeInfo NodeInfo) *Status
}

var _ FilterPlugin = FilterFunc(nil)

// FilterFunc adapts an ordinary function to a FilterPlugin.
type FilterFunc func(pod Pod, nodeInfo NodeInfo) *Status

// Filter returns f(pod, nodeInfo).
func (f FilterFunc) Filter(pod Pod, nodeInfo NodeInfo) *Status {
	return f(pod, nodeInfo)
}

// ScorePlugin is a WebAssembly implementation of framework.ScorePlugin.
//
// Note: This is int32, not int64. See /RATIONALE.md for why.
type ScorePlugin interface {
	Score(pod Pod, nodeName string) (int32, *Status)
}

var _ ScorePlugin = ScoreFunc(nil)

// ScoreFunc adapts an ordinary function to a ScorePlugin.
type ScoreFunc func(pod Pod, nodeName string) (int32, *Status)

// Score returns f(pod, nodeName).
func (f ScoreFunc) Score(pod Pod, nodeName string) (int32, *Status) {
	return f(pod, nodeName)
}

type NodeInfo interface {
	Node() *protoapi.Node
}

type Pod interface {
	Metadata() *meta.ObjectMeta
	Spec() *protoapi.PodSpec
	Status() *protoapi.PodStatus
}
