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
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

func main() {
	score.Plugin = api.ScoreFunc(score100IfNameEqualsPodSpec)
}

// score100IfNameEqualsPodSpec scores 100 if this node name equals its pod spec.
func score100IfNameEqualsPodSpec(pod api.Pod, nodeName string) (int32, *api.Status) {
	podSpecNodeName := nilToEmpty(pod.Spec().NodeName)
	if nodeName == podSpecNodeName {
		return 100, nil
	}
	return 0, nil
}

func nilToEmpty(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}
