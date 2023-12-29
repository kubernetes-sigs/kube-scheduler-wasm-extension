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

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/examples/advanced/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	klog "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

// main is compiled to an exported Wasm function named "_start", called by the
// Wasm scheduler plugin during initialization.
func main() {
	enqueue.SetPlugin(func(klog klog.Klog, jsonConfig []byte, h handleapi.Handle) api.EnqueueExtensions {
		p := pluginInitializer(klog, jsonConfig, h)
		return p.(api.EnqueueExtensions)
	})
	prescore.SetPlugin(func(klog klog.Klog, jsonConfig []byte, h handleapi.Handle) api.PreScorePlugin {
		p := pluginInitializer(klog, jsonConfig, h)
		return p.(api.PreScorePlugin)
	})
	score.SetPlugin(func(klog klog.Klog, jsonConfig []byte, h handleapi.Handle) api.ScorePlugin {
		p := pluginInitializer(klog, jsonConfig, h)
		return p.(api.ScorePlugin)
	})
}

func pluginInitializer(klog klog.Klog, jsonConfig []byte, h handleapi.Handle) api.Plugin {
	// The plugin package uses only normal Go code, which allows it to be
	// unit testable via `tinygo test -target=wasi` as well normal `go test`.
	//
	// The real implementations use Wasm host functions
	// (go:wasmimport), which cannot be tested with `tinygo test -target=wasi`.
	plugin, err := plugin.New(klog, jsonConfig, h)
	if err != nil {
		panic(err)
	}
	return plugin
}
