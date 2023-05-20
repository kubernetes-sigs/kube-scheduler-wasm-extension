package plugin

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/internal"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/vfs"
)

// New initializes a new plugin and returns it.
func New(runtime.Object, framework.Handle) (pl framework.Plugin, err error) {
	// TODO: make this configuration via URL
	const guestPath = "testdata/vfsmain/main.wasm"
	const guestName = "example"

	ctx := context.Background()

	runtime, module, err := internal.CompileGuest(ctx, guestPath)
	if err != nil {
		return nil, err
	}

	if vfs.IsVFSPlugin(module) {
		return vfs.NewPlugin(runtime, module, guestName)
	}
	panic("TODO: ABIPlugin")
}
