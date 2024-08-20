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

package imports

import (
	"encoding/json"
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
)

// StatusToCode returns a WebAssembly compatible result for the input status,
// after sending any reason to the host.
func StatusToCode(s *api.Status) uint32 {
	// Nil status is the same as one with a success code.
	if s == nil || s.Code == api.StatusCodeSuccess {
		return uint32(api.StatusCodeSuccess)
	}

	// WebAssembly Core 2.0 (DRAFT) only includes numeric types. Return the
	// reason using a host function.
	if reason := s.Reason; reason != "" {
		setStatusReason(reason)
	}
	return uint32(s.Code)
}

// setStatusReason overwrites the status reason
func setStatusReason(reason string) {
	ptr, size := mem.StringToPtr(reason)
	k8sSchedulerResultStatusReason(ptr, size)
	runtime.KeepAlive(reason) // until ptr is no longer needed.
}

func CurrentNodeName() string {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	return mem.GetString(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerCurrentNodeName(ptr, limit)
	})
}

func NodeImageStates(nodeName string) map[string]*api.ImageStateSummary {
	nodenamePtr, nodenameSize := mem.StringToPtr(nodeName)
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	jsonStr := mem.GetString(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerNodeImageStates(nodenamePtr, nodenameSize, ptr, limit)
	})
	byte := []byte(jsonStr)
	var nodeImageStates map[string]*api.ImageStateSummary
	err := json.Unmarshal(byte, &nodeImageStates)
	if err != nil {
		panic(err)
	}
	return nodeImageStates
}

func Node(nodeName string, updater func([]byte) error) error {
	nodenamePtr, nodenameSize := mem.StringToPtr(nodeName)
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	return mem.Update(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sApiNode(nodenamePtr, nodenameSize, ptr, limit)
	}, updater)
}

func Nodes(updater func([]byte) error) error {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	return mem.Update(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sApiNodeList(ptr, limit)
	}, updater)
}

func CurrentPod(updater func([]byte) error) error {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	return mem.Update(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerCurrentPod(ptr, limit)
	}, updater)
}

func TargetPod(updater func([]byte) error) error {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	return mem.Update(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerTargetPod(ptr, limit)
	}, updater)
}

func NodeToStatusMap() map[string]api.StatusCode {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	jsonStr := mem.GetString(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerNodeToStatusMap(ptr, limit)
	})
	byte := []byte(jsonStr)
	var nodeToMap map[string]api.StatusCode
	err := json.Unmarshal(byte, &nodeToMap)
	if err != nil {
		panic(err)
	}
	return nodeToMap
}

// normalizeScore calls NodeScoreList
func NodeScoreList() map[string]int {
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	jsonStr := mem.GetString(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sSchedulerNodeScoreList(ptr, limit)
	})
	byte := []byte(jsonStr)
	var nodeScoreList map[string]int
	err := json.Unmarshal(byte, &nodeScoreList)
	if err != nil {
		panic(err)
	}
	return nodeScoreList
}
