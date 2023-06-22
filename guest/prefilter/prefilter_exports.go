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

package prefilter

import (
	"runtime"
	"unsafe"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/types"
)

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = prefilter

// prefilter is only exported to the host.
//
//export prefilter
func prefilter() uint32 { //nolint
	if Plugin == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("PreFilter imported, but PreFilter.Plugin nil")
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	nodeNames, status := Plugin.PreFilter(&types.Pod{})

	// If plugin returned nodeNames, concatenate them into a C-string and call
	// the host with the count and memory region.
	cString := toNULTerminated(nodeNames)
	if cString != nil {
		ptr := uint32(uintptr(unsafe.Pointer(&cString[0])))
		size := uint32(len(cString))
		setNodeNamesResult(ptr, size)
		runtime.KeepAlive(cString) // until ptr is no longer needed.
	}

	return imports.StatusToCode(status)
}
