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

// Package prescore exports an api.PreScorePlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package handle

import (
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
)

func RejectWaitingPod(uid string) bool {
	ptr, size := mem.StringToPtr(uid)

	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	wasmBool := mem.SendAndGetUint64(ptr, size, func(input_ptr, input_size, ptr uint32, limit mem.BufLimit) {
		rejectWaitingPod(input_ptr, input_size, ptr, limit)
	})
	runtime.KeepAlive(uid)
	return wasmBool == 1
}

func GetWaitingPod(uid string) api.WaitingPod {
	ptr, size := mem.StringToPtr(uid)

	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	mem.SendAndGetString(ptr, size, func(input_ptr, input_size, ptr uint32, limit mem.BufLimit) {
		getWaitingPod(input_ptr, input_size, ptr, limit)
	})
	runtime.KeepAlive(uid)

	waitingPod := make([]api.WaitingPod, size)
	return waitingPod[0]
}
