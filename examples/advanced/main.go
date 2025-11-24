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
// 'tinygo build -target=wasi'. See /guest/RATIONALE.md for details.
package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/examples/advanced/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/eventrecorder"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

// main is compiled to an exported Wasm function named "_start", called by the
// Wasm scheduler plugin during initialization.
func main() {
	// The plugin package uses only normal Go code, which allows it to be
	// unit testable via `tinygo test -target=wasi` as well normal `go test`.
	//
	// The real implementations, such as `config.Get()` use Wasm host functions
	// (go:wasmimport), which cannot be tested with `tinygo test -target=wasi`.
	plugin, err := plugin.New(klog.Get(), config.Get(), eventrecorder.Get())
	if err != nil {
		panic(err)
	}
	// Instead of using `plugin.Set`, this configures only the interfaces
	// implemented by the plugin. The Wasm host only calls functions imported,
	// so this prevents additional overhead.
	enqueue.SetPlugin(plugin)
	prescore.SetPlugin(plugin)
	score.SetPlugin(plugin)
}
