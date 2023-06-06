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
	"os"
	"sync/atomic"
	"time"

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

	return NewFromConfig(context.Background(), config)
}

// NewFromConfig is like New, except it allows us to explicitly provide the
// context and configuration of the plugin. This allows flexibility in tests.
func NewFromConfig(ctx context.Context, config PluginConfig) (framework.Plugin, error) {
	guestBin, err := os.ReadFile(config.GuestPath)
	if err != nil {
		return nil, fmt.Errorf("wasm: error reading guest binary at %s: %w", config.GuestPath, err)
	}

	runtime, guestModule, err := prepareRuntime(ctx, guestBin)
	if err != nil {
		return nil, err
	}

	guestName := config.GuestName
	if guestName == "" {
		guestName = guestModule.Name()
	}

	pl := &wasmPlugin{
		runtime:           runtime,
		guestName:         guestName,
		guestModule:       guestModule,
		guestModuleConfig: wazero.NewModuleConfig(),
		instanceCounter:   atomic.Uint64{},
	}

	pl.pool, err = newGuestPool(ctx, pl.newGuest)
	if err != nil {
		return nil, fmt.Errorf("failed to create a guest pool: %w", err)
	}

	return pl, nil
}

type wasmPlugin struct {
	runtime           wazero.Runtime
	guestName         string
	guestModule       wazero.CompiledModule
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
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	// TODO: support AddPod in wasm guest.
	return nil
}

// RemovePod implements the same method as documented on framework.PreFilterExtensions.
func (pl *wasmPlugin) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	// TODO: support RemovePod in wasm guest.
	return nil
}

var _ framework.PreFilterPlugin = (*wasmPlugin)(nil)

// PreFilterExtensions implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// PreFilter implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	// PreFilter is the first stage in scheduling. If there's an existing guest
	// association, unassign it, so that we can make a pod-specific one.
	pl.pool.unassignForScheduling()

	_, err := pl.pool.getOrCreateGuest(ctx, pod.GetUID())
	if err != nil {
		return nil, framework.AsStatus(err)
	}

	// TODO: support PreFilter in wasm guest.

	return nil, nil
}

var _ framework.FilterPlugin = (*wasmPlugin)(nil)

// Filter implements the same method as documented on framework.FilterPlugin.
func (pl *wasmPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	g, err := pl.pool.getOrCreateGuest(ctx, pod.GetUID())
	if err != nil {
		return framework.AsStatus(err)
	}

	// The guest calls host functions to access the pod and node data. Add
	// this to the context, so that they are accessible.
	params := &filterParams{pod: pod, nodeInfo: nodeInfo}
	ctx = context.WithValue(ctx, filterParamsKey{}, params)
	return g.filter(ctx)
}

var _ framework.PostFilterPlugin = (*wasmPlugin)(nil)

// PostFilter implements the same method as documented on framework.PostFilterPlugin.
func (pl *wasmPlugin) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	panic("TODO: PostFilter")
}

var _ framework.PreScorePlugin = (*wasmPlugin)(nil)

// PreScore implements the same method as documented on framework.PreScorePlugin.
func (pl *wasmPlugin) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	panic("TODO: PreScore")
}

var _ framework.ScoreExtensions = (*wasmPlugin)(nil)

// NormalizeScore implements the same method as documented on framework.ScoreExtensions.
func (pl *wasmPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	panic("TODO: PreScore")
}

var _ framework.ScorePlugin = (*wasmPlugin)(nil)

// Score implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	panic("TODO: Score")
}

// ScoreExtensions implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) ScoreExtensions() framework.ScoreExtensions {
	panic("TODO: Score")
}

var _ framework.ReservePlugin = (*wasmPlugin)(nil)

// Reserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Reserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) *framework.Status {
	// TODO: support Reserve in wasm guest.
	// Currently, it's implemented to implement the ReservePlugin interface.
	return nil
}

// Unreserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Unreserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	pl.pool.unassignForBinding(p.GetUID())
	// TODO: support Unreserve in wasm guest.
}

var _ framework.PreBindPlugin = (*wasmPlugin)(nil)

// PreBind implements the same method as documented on framework.PreBindPlugin.
func (pl *wasmPlugin) PreBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	panic("TODO: PreBind")
}

var _ framework.PostBindPlugin = (*wasmPlugin)(nil)

// PostBind implements the same method as documented on framework.PostBindPlugin.
func (pl *wasmPlugin) PostBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	pl.pool.unassignForBinding(pod.GetUID())
	// TODO: support PostBind in wasm guest.
}

var _ framework.PermitPlugin = (*wasmPlugin)(nil)

// Permit implements the same method as documented on framework.PermitPlugin.
func (pl *wasmPlugin) Permit(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	// assume that the pod is going to binding cycle and continue to assign the instance to the pod.
	// unassign the instance in Unreserve or PostBind.
	pl.pool.assignForBinding(p.GetUID())

	// TODO: support Permit in wasm guest.

	return nil, 0
}

var _ framework.BindPlugin = (*wasmPlugin)(nil)

// Bind implements the same method as documented on framework.BindPlugin.
func (pl *wasmPlugin) Bind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	panic("TODO: Bind")
}

// Close implements io.Closer
func (pl *wasmPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}
