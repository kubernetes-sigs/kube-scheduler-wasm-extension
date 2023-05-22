package plugin

import (
	"context"

	"k8s.io/kubernetes/pkg/scheduler/framework"

	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/abi"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/internal"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/vfs"
)

// New initializes a new plugin and returns it.
func New(guestPath string /*runtime.Object, framework.Handle*/) (pl framework.Plugin, err error) {
	// TODO: make this configuration via URL
	const guestName = "example"

	ctx := context.Background()

	runtime, module, err := internal.CompileGuest(ctx, guestPath)
	if err != nil {
		return nil, err
	}

	if abi.IsABIPlugin(module) {
		return abi.NewPlugin(ctx, runtime, module, guestName)
	}
	if vfs.IsVFSPlugin(module) {
		return vfs.NewPlugin(ctx, runtime, module, guestName)
	}
	panic("unexpected")
}
