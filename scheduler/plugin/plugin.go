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

package wasm

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

const PluginName = "wasm"

var _ frameworkruntime.PluginFactory = New

// New initializes a new plugin and returns it.
func New(configuration runtime.Object, frameworkHandle framework.Handle) (framework.Plugin, error) {
	config := PluginConfig{}
	if err := frameworkruntime.DecodeInto(configuration, &config); err != nil {
		return nil, fmt.Errorf("failed to decode into %s PluginConfig: %w", PluginName, err)
	}

	plugin, err := NewFromConfig(context.Background(), config)
	return maskInterfaces(plugin), err
}

// maskInterfaces ensures the caller can do type checking to detect what the
// plugin supports.
func maskInterfaces(plugin *wasmPlugin) framework.Plugin {
	if plugin == nil {
		return nil
	}
	switch plugin.guestExports {
	case exportFilterPlugin:
		return struct {
			framework.FilterPlugin
			io.Closer
		}{plugin, plugin}
	case exportScorePlugin:
		return struct {
			framework.ScorePlugin
			io.Closer
		}{plugin, plugin}
	case exportFilterPlugin | exportScorePlugin:
		type filterScore interface {
			framework.FilterPlugin
			framework.ScorePlugin
			io.Closer
		}
		return struct{ filterScore }{plugin}
	}
	panic("BUG: unhandled exports")
}

// NewFromConfig is like New, except it allows us to explicitly provide the
// context and configuration of the plugin. This allows flexibility in tests.
func NewFromConfig(ctx context.Context, config PluginConfig) (*wasmPlugin, error) {
	guestBin, err := os.ReadFile(config.GuestPath)
	if err != nil {
		return nil, fmt.Errorf("wasm: error reading guest binary at %s: %w", config.GuestPath, err)
	}

	runtime, guestModule, err := prepareRuntime(ctx, guestBin)
	if err != nil {
		return nil, err
	}

	pl, err := newWasmPlugin(ctx, runtime, guestModule, config)
	if err != nil {
		_ = runtime.Close(ctx)
	}
	return pl, err
}

// newWasmPlugin is extracted to prevent small bugs: The caller must close the
// wazero.Runtime to avoid leaking mmapped files.
func newWasmPlugin(ctx context.Context, runtime wazero.Runtime, guestModule wazero.CompiledModule, config PluginConfig) (*wasmPlugin, error) {
	var guestExports exports
	var err error
	if guestExports, err = detectExports(guestModule.ExportedFunctions()); err != nil {
		return nil, err
	} else if guestExports == 0 {
		return nil, fmt.Errorf("wasm: guest doesn't export plugin functions")
	}

	guestName := config.GuestName
	if guestName == "" {
		guestName = guestModule.Name()
	}

	pl := &wasmPlugin{
		runtime:           runtime,
		guestName:         guestName,
		guestModule:       guestModule,
		guestExports:      guestExports,
		guestModuleConfig: wazero.NewModuleConfig(),
		instanceCounter:   atomic.Uint64{},
	}

	if pl.pool, err = newGuestPool(ctx, pl.newGuest); err != nil {
		return nil, fmt.Errorf("failed to create a guest pool: %w", err)
	}
	return pl, nil
}

type wasmPlugin struct {
	runtime           wazero.Runtime
	guestName         string
	guestModule       wazero.CompiledModule
	guestExports      exports
	guestModuleConfig wazero.ModuleConfig
	instanceCounter   atomic.Uint64
	pool              *guestPool[*guest]
}

var _ framework.Plugin = (*wasmPlugin)(nil)

// Name implements the same method as documented on framework.Plugin.
func (pl *wasmPlugin) Name() string {
	return PluginName
}

var _ framework.PreFilterExtensions = (*wasmPlugin)(nil)

// AddPod implements the same method as documented on framework.PreFilterExtensions.
func (pl *wasmPlugin) AddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	panic("TODO: scheduling: AddPod")
}

// RemovePod implements the same method as documented on framework.PreFilterExtensions.
func (pl *wasmPlugin) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	panic("TODO: scheduling: RemovePod")
}

var _ framework.PreFilterPlugin = (*wasmPlugin)(nil)

// PreFilterExtensions implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	panic("TODO: scheduling: PreFilterExtensions")
}

// PreFilter implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod) (result *framework.PreFilterResult, status *framework.Status) {
	if err := pl.pool.doWithSchedulingGuest(ctx, cycleID(pod), func(g *guest) {
		// TODO: partially implemented for testing
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.FilterPlugin = (*wasmPlugin)(nil)

// Filter implements the same method as documented on framework.FilterPlugin.
func (pl *wasmPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) (status *framework.Status) {
	// Add the params to the go context so that the corresponding host function
	// can look them up.
	params := &params{pod: pod, nodeInfo: nodeInfo}
	ctx = context.WithValue(ctx, paramsKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, cycleID(pod), func(g *guest) {
		status = g.filter(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.PostFilterPlugin = (*wasmPlugin)(nil)

// PostFilter implements the same method as documented on framework.PostFilterPlugin.
func (pl *wasmPlugin) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	panic("TODO: scheduling: PostFilter")
}

var _ framework.PreScorePlugin = (*wasmPlugin)(nil)

// PreScore implements the same method as documented on framework.PreScorePlugin.
func (pl *wasmPlugin) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	panic("TODO: scheduling: PreScore")
}

var _ framework.ScoreExtensions = (*wasmPlugin)(nil)

// NormalizeScore implements the same method as documented on framework.ScoreExtensions.
func (pl *wasmPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	panic("TODO: scheduling: NormalizeScore")
}

var _ framework.ScorePlugin = (*wasmPlugin)(nil)

// Score implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (score int64, status *framework.Status) {
	// Add the params to the go context so that the corresponding host function
	// can look them up.
	params := &params{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, paramsKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, cycleID(pod), func(g *guest) {
		score, status = g.score(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

// ScoreExtensions implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) ScoreExtensions() framework.ScoreExtensions {
	panic("TODO: scheduling: ScoreExtensions")
}

var _ framework.ReservePlugin = (*wasmPlugin)(nil)

// Reserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (status *framework.Status) {
	if err := pl.pool.doWithSchedulingGuest(ctx, cycleID(pod), func(g *guest) {
		// TODO: partially implemented for testing
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

// Unreserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	// Note: Unlike the below diagram, this is not a part of the scheduling
	// cycle, rather the binding on error.
	// https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#extension-points

	cycleID := cycleID(pod)
	defer pl.pool.freeFromBinding(cycleID) // the cycle is over, put it back into the pool.

	// TODO: partially implemented for testing
}

var _ framework.PreBindPlugin = (*wasmPlugin)(nil)

// PreBind implements the same method as documented on framework.PreBindPlugin.
func (pl *wasmPlugin) PreBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	panic("TODO: binding: PreBind")
}

var _ framework.PostBindPlugin = (*wasmPlugin)(nil)

// PostBind implements the same method as documented on framework.PostBindPlugin.
func (pl *wasmPlugin) PostBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	cycleID := cycleID(pod)
	defer pl.pool.freeFromBinding(cycleID) // the cycle is over, put it back into the pool.

	// TODO: partially implemented for testing
}

var _ framework.PermitPlugin = (*wasmPlugin)(nil)

// Permit implements the same method as documented on framework.PermitPlugin.
func (pl *wasmPlugin) Permit(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	_ = pl.pool.getForBinding(cycleID(pod))

	// TODO: partially implemented for testing

	return nil, 0
}

var _ framework.BindPlugin = (*wasmPlugin)(nil)

// Bind implements the same method as documented on framework.BindPlugin.
func (pl *wasmPlugin) Bind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	panic("TODO: binding: Bind")
}

// Close implements io.Closer
func (pl *wasmPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}

// cycleID is stable through a scheduling or binding cycle. For example, it
// will be different when the same pod is rescheduled due to an error. The
// cycleID is not derived from the v1.Pod UID for this reason.
//
// We use the last 32-bits of the pod's pointer as its ID, as the struct is
// re-instantiated each scheduling cycle, but the same object is used for each
// callback within one.
// See https://github.com/kubernetes/kubernetes/blob/9740bc0e0a10aad753cf7fcbed0c7be25ab200dd/pkg/scheduler/schedule_one.go#L133
func cycleID(pod *v1.Pod) uint32 {
	podPtr := uintptr(unsafe.Pointer(pod))
	return uint32(podPtr)
}
