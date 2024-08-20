//go:build tinygo.wasm

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

import "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"

//go:wasmimport k8s.io/api node
func k8sApiNode(uint32, uint32, uint32, mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/api nodeList
func k8sApiNodeList(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler nodeToStatusMap
func k8sSchedulerNodeToStatusMap(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler result.status_reason
func k8sSchedulerResultStatusReason(ptr, size uint32)

//go:wasmimport k8s.io/scheduler nodeScoreList
func k8sSchedulerNodeScoreList(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler currentNodeName
func k8sSchedulerCurrentNodeName(uint32, mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler currentPod
func k8sSchedulerCurrentPod(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler targetPod
func k8sSchedulerTargetPod(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport k8s.io/scheduler nodeImageStates
func k8sSchedulerNodeImageStates(uint32, uint32, uint32, mem.BufLimit) (len uint32)
