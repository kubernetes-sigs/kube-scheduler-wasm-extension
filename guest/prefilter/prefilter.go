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

// Package prefilter exports an api.PreFilterPlugin to the host. Only import
// this package when setting Plugin, as doing otherwise will cause overhead.
package prefilter

// TODO: The guest should always implement PreFilter, so it can know to
// reset state when the same pod has been re-scheduled due to an error. We
// need to both implement and test this.

import "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"

// Plugin should be assigned in `main` to an api.PreFilterPlugin instance.
//
// For example:
//
//	func main() {
//		filter.Plugin = api.PreFilterPlugin(podSpecName)
//	}
var Plugin api.PreFilterPlugin
