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

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	"os"

	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

func main() {
	// Multiple tests are here to reduce re-compilation time and size checked
	// into git.
	if len(os.Args) == 2 && os.Args[1] == "score100IfNameEqualsPodSpec" {
		score.SetPlugin(score100IfNameEqualsPodSpec{})
	} else {
		score.SetPlugin(noop{})
	}
}

// noop doesn't do anything, except evaluate each parameter.
type noop struct{}

func (noop) Score(state api.CycleState, pod proto.Pod, nodeName string) (score int32, status *api.Status) {
	_, _ = state.Read("ok")
	_ = pod.Spec()
	_ = nodeName
	return
}

// score100IfNameEqualsPodSpec returns 100 if a node name equals its pod spec.
type score100IfNameEqualsPodSpec struct{}

func (score100IfNameEqualsPodSpec) Score(_ api.CycleState, pod proto.Pod, nodeName string) (int32, *api.Status) {
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
