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
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
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
		statusReason(reason)
	}
	return uint32(s.Code)
}

// statusReason overwrites the status reason
func statusReason(reason string) {
	ptr, size := stringToPtr(reason)
	k8sSchedulerStatusReason(ptr, size)
	runtime.KeepAlive(reason) // keep reason alive until ptr is no longer needed.
}

func NodeName() string {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return getString(func(ptr uint32, limit bufLimit) (len uint32) {
		return k8sApiNodeName(ptr, limit)
	})
}

func NodeInfoNode(updater func([]byte) error) error {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return update(func(ptr uint32, limit bufLimit) (len uint32) {
		return k8sApiNodeInfoNode(ptr, limit)
	}, updater)
}

func Pod(updater func([]byte) error) error {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return update(func(ptr uint32, limit bufLimit) (len uint32) {
		return k8sApiPod(ptr, limit)
	}, updater)
}
