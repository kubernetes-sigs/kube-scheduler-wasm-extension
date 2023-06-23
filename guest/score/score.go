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

// Package score exports an api.ScorePlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package score

import "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"

// SetPlugin should be called in `main` to assign an api.ScorePlugin instance.
//
// For example:
//
//	func main() {
//		score.SetPlugin(score100IfNameEqualsPodSpec)
//	}
func SetPlugin(scorePlugin api.ScorePlugin) {
	if scorePlugin == nil {
		panic("nil scorePlugin")
	}
	plugin = scorePlugin
}
