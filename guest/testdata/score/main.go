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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/scoreextensions"
)

type extensionPoints interface {
	api.PreScorePlugin
	api.ScorePlugin
	api.ScoreExtensions
}

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	var plugin extensionPoints = noopPlugin{}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "score":
			plugin = scorePlugin{}
		case "preScore":
			plugin = preScorePlugin{}
		case "scoreExtensions":
			plugin = scoreExtensions{}
		}
	}
	prescore.SetPlugin(plugin)
	score.SetPlugin(plugin)
	scoreextensions.SetPlugin(plugin)
}

// noopPlugin doesn't do anything, except evaluate each parameter.
type noopPlugin struct{}

func (noopPlugin) PreScore(state api.CycleState, pod proto.Pod, nodeList proto.NodeList) *api.Status {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeList.Items()
	return nil
}

func (noopPlugin) Score(state api.CycleState, pod proto.Pod, nodeName string) (score int32, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeName
	return
}

func (noopPlugin) NormalizeScore(state api.CycleState, pod proto.Pod, scores api.NodeScore) (updatedScores map[string]int, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = scores
	return
}

// preScorePlugin returns the count of the node list as the status
type preScorePlugin struct{ noopPlugin }

func (preScorePlugin) PreScore(_ api.CycleState, _ proto.Pod, nodeList proto.NodeList) *api.Status {
	return &api.Status{Code: api.StatusCode(len(nodeList.Items()))}
}

// scorePlugin returns 100 if a node name equals its pod spec.
type scorePlugin struct{ noopPlugin }

func (scorePlugin) Score(_ api.CycleState, pod proto.Pod, nodeName string) (int32, *api.Status) {
	podSpecNodeName := pod.Spec().GetNodeName()
	if nodeName == podSpecNodeName {
		return 100, nil
	}
	return 0, nil
}

type scoreExtensions struct{ noopPlugin }

// NormalizeScore multiply score by 100.
func (scoreExtensions) NormalizeScore(_ api.CycleState, pod proto.Pod, scores api.NodeScore) (map[string]int, *api.Status) {
	updatedScoreMap := make(map[string]int)
	scoreMap := scores.Map()
	for nodeName, score := range scoreMap {
		updatedScoreMap[nodeName] = score * 100
	}
	return updatedScoreMap, nil
}
