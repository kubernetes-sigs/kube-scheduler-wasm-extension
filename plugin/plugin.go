package plugin

import (
	"context"
	"sync"
	"sync/atomic"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/tetratelabs/wazero"
)

const (
	PluginName = "wasm"
)

// New initializes a new plugin and returns it.
func New(guestPath string /*runtime.Object, framework.Handle*/) (framework.Plugin, error) {
	// TODO: make this configuration via URL
	const guestName = "example"

	ctx := context.Background()

	runtime, guestModule, err := prepareRuntime(ctx, guestPath)
	if err != nil {
		return nil, err
	}

	pl := &wasmPlugin{
		guestModuleConfig: wazero.NewModuleConfig(),
		guestName:         guestName,
		runtime:           runtime,
		guestModule:       guestModule,
		instanceCounter:   atomic.Uint64{},
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	g, err := pl.getOrCreateGuest(ctx)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, err
	}
	pl.pool.Put(g)

	return pl, nil
}

type wasmPlugin struct {
	runtime           wazero.Runtime
	guestName         string
	guestModule       wazero.CompiledModule
	guestModuleConfig wazero.ModuleConfig
	instanceCounter   atomic.Uint64
	pool              sync.Pool
}

var _ framework.FilterPlugin = (*wasmPlugin)(nil)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *wasmPlugin) Name() string {
	return PluginName
}

func (pl *wasmPlugin) getOrCreateGuest(ctx context.Context) (*guest, error) {
	poolG := pl.pool.Get()
	if poolG == nil {
		if g, createErr := pl.newGuest(ctx); createErr != nil {
			return nil, createErr
		} else {
			poolG = g
		}
	}
	return poolG.(*guest), nil
}

// Filter invoked at the filter extension point.
func (pl *wasmPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	g, err := pl.getOrCreateGuest(ctx)
	if err != nil {
		return framework.AsStatus(err)
	}
	defer pl.pool.Put(g)

	// The guest Wasm may call host functions, so we add context parameters of
	// the current args.
	args := &filterArgs{pod: pod, nodeInfo: nodeInfo}
	ctx = context.WithValue(ctx, filterArgsKey{}, args)
	return g.filter(ctx)
}

// Close implements io.Closer
func (pl *wasmPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}
