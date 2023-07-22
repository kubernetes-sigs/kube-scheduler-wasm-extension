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

// Package main is the entrypoint of the %.wasm file, compiled with
// '-target=wasi'. See /guest/RATIONALE.md for details.
package main

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/examples/nodenumber/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

// main is compiled to a WebAssembly function named "_start", called by the
// wasm scheduler plugin during initialization.
func main() {
	plugin := &plugin.NodeNumber{}
	// Below is like `var _ api.EnqueueExtensions = plugin`, except it also
	// wires up functions the host should provide (go:wasmimport).
	enqueue.SetPlugin(plugin)
	prescore.SetPlugin(plugin)
	score.SetPlugin(plugin)
}
