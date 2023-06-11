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

package filter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/types"
)

// filter is only exported to the host.
//
//export filter
func filter() uint32 { //nolint
	if Plugin == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("filter imported, but filter.Plugin nil")
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	// TODO: Make these fields and reset on pre-filter or similar.
	s := Plugin.Filter(&types.Pod{}, &types.NodeInfo{})
	return imports.StatusToCode(s)
}
