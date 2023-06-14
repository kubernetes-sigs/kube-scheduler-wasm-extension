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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

func main() {
	// These plugins don't do anything, except evaluate each parameter. This
	// helps show if caching works.
	filter.Plugin = api.FilterFunc(filterNoop)
	score.Plugin = api.ScoreFunc(scoreNoop)
}

func filterNoop(pod api.Pod, nodeInfo api.NodeInfo) (status *api.Status) {
	_ = pod.Spec()
	_ = nodeInfo.Node()
	return
}

func scoreNoop(pod api.Pod, nodeName string) (score int32, status *api.Status) {
	_ = pod.Spec()
	_ = nodeName
	return
}
