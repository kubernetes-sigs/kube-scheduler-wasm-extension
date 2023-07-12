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

// Package prefilter exports an api.PreFilterPlugin to the host.
package prefilter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	internalprefilter "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/prefilter"
)

// SetPlugin should be called in `main` to assign an api.PreFilterPlugin
// instance.
//
// For example:
//
//	func main() {
//		prefilter.SetPlugin(podSpecName{})
//	}
//
//	type podSpecName struct{}
//
//	func (podSpecName) PreFilter(state api.CycleState, pod api.Pod) (nodeNames []string, status *api.Status) {
//		panic("implement me")
//	}
func SetPlugin(prefilterPlugin api.PreFilterPlugin) {
	internalprefilter.SetPlugin(prefilterPlugin)
}
