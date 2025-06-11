/*
Copyright 2019 The Kubernetes Authors.

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

package scheduler_perf

import (
	"testing"

	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	perf "k8s.io/kubernetes/test/integration/scheduler_perf"

	nodenumberplugin "sigs.k8s.io/kube-scheduler-wasm-extension/internal/e2e/scheduler_perf/nodenumber"
	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
)

var (
	registory frameworkruntime.Registry = frameworkruntime.Registry{
		nodenumberplugin.Name: nodenumberplugin.New,
		pluginName:            wasm.PluginFactory(pluginName),
	}
	pluginName = "wasm"
)

func BenchmarkPerfScheduling(b *testing.B) {
	// Use relative path from current directory, following upstream convention
	perf.RunBenchmarkPerfScheduling(b, "config/performance-config.yaml", "wasm", registory)
}
