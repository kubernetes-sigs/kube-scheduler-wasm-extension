package scheduler_perf

import (
	"testing"

	scheduler_perf "github.com/sanposhiho/kubernetes/test/integration/scheduler_perf"
	"k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
)

func BenchmarkPerfScheduling(b *testing.B) {
	scheduler_perf.RunBenchmarkPerfScheduling(b, runtime.Registry{wasm.PluginName: wasm.New})
}
