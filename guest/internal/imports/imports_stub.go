//go:build !tinygo.wasm

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

// k8sApiNodeInfoNode is stubbed for compilation outside TinyGo.
func k8sApiNodeInfoNode(uint32, bufLimit) (len uint32) { return }

// k8sApiNodeName is stubbed for compilation outside TinyGo.
func k8sApiNodeName(uint32, bufLimit) (len uint32) { return }

// k8sApiPod is stubbed for compilation outside TinyGo.
func k8sApiPod(uint32, bufLimit) (len uint32) { return }

// k8sSchedulerStatusReason is stubbed for compilation outside TinyGo.
func k8sSchedulerStatusReason(uint32, uint32) {}
