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

// Package filter exports an api.FilterPlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package filter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/types"
)

// Plugin should be assigned in `main` to an api.FilterPlugin instance.
//
// For example:
//
//	func main() {
//		filter.Plugin = api.FilterFunc(nameEqualsPodSpec)
//	}
var Plugin api.FilterPlugin

// filter is only exported to the host.
//
//go:export filter
func filter() uint32 { //nolint
	// Pass on unconfigured filter
	if Plugin == nil {
		return uint32(api.StatusCodeSuccess)
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	// TODO: Make these fields and reset on pre-filter or similar.
	s := Plugin.Filter(&types.Pod{}, &types.NodeInfo{})
	return imports.StatusToCode(s)
}
