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

package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

// main is compiled to an exported Wasm function named "_start", called by the
// Wasm scheduler plugin during initialization.
func main() {}
func init() {
	// The plugin package uses only normal Go code, which allows it to be
	// unit testable via `tinygo test -target=wasi` as well normal `go test`.
	//
	// The real implementations, such as `config.Get()` use Wasm host functions
	// (go:wasmimport), which cannot be tested with `tinygo test -target=wasi`.
	plugin := &imageLocality{
		sharedLister: sharedlister.Get(),
	}
	// Instead of using `plugin.Set`, this configures only the interfaces
	// implemented by the plugin. The Wasm host only calls functions imported,
	// so this prevents additional overhead.
	score.SetPlugin(plugin)
}
