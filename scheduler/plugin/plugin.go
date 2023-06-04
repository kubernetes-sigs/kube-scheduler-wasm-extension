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
	"sync"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"

	"github.com/tetratelabs/wazero"
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

	pl := &wasmPlugin{
		guestModuleConfig: wazero.NewModuleConfig(),
		guestName:         config.GuestName,
		runtime:           runtime,
		guestModule:       guestModule,
		instanceCounter:   atomic.Uint64{},
	}

	pl.pool, err = pl.newGuestPool(ctx)
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
	pool              *guestPool
}

// guestPool manages the wasm guest instances.
// It remember which guest instance is assigned to which pod in scheduling cycle or binding cycle.
type guestPool struct {
	// pool is a pool of unused guest instances.
	pool sync.Pool

	// scheduledPodUID is the UID of the pod that is being scheduled.
	// The plugin will update this field at the beginning of each scheduling cycle (PreFilter).
	//
	// Note: We cannot unassign the guest instance in PostFilter/Permit
	// because PostFilter is not enough to notice failures in the scheduling cycle,
	// it's called only when the Pod is rejected in PreFilter or Filter.
	// So, we need to trach Pods in the scheduling cycle and the binding cycle separately
	// so that we can know which guest instance to unassign at PreFilter.
	schedulingPodUID types.UID
	// assignedToSchedulingPod has the guest instance assignedToSchedulingPod to the pod.
	// assignedToSchedulingPod instance won't be put back to the pool until the scheduling cycle of this Pod is finished.
	// The plugin will update this field at the beginning of each scheduling cycle (PreFilter) along with schedulingPodUID.
	assignedToSchedulingPod *guest
	// assignedToSchedulingPodLock is a lock to protect the access to the assigned instance.
	// The scheduler may call Filter(), AddPod(), RemovePod() concurrently, and we need to take a lock.
	// But, other methods of the plugin should be invoked for the same Pod serially.
	assignedToSchedulingPodLock sync.Mutex

	// assignedToBindingPod has the guest instances for Pods in binding cycle.
	assignedToBindingPod map[types.UID]*guest
}

func (pl *wasmPlugin) newGuestPool(ctx context.Context) (*guestPool, error) {
	p := &guestPool{
		assignedToBindingPod: make(map[types.UID]*guest),
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	g, createErr := pl.newGuest(ctx)
	if createErr != nil {
		return nil, createErr
	}
	p.put(g)

	return p, nil
}

// assignForScheduling assigns the guest instance to the pod in the scheduling cycle
// so that the same pod can always get the same guest instance.
func (g *guestPool) assignForScheduling(guest *guest, podUID types.UID) {
	g.assignedToSchedulingPod = guest
	g.schedulingPodUID = podUID
}

// assignForBinding assigns the guest instance to the pod in the binding cycle.
// This function doesn't take a lock because it is supposed to be called in the scheduling cycle meaning it'll never called in parallel.
func (g *guestPool) assignForBinding(podUID types.UID) {
	guest := g.assignedToSchedulingPod
	g.assignedToBindingPod[podUID] = guest
}

// unassignForScheduling unassigns the guest instance from the pod and put the instance back to the pool.
func (g *guestPool) unassignForScheduling() {
	assigned := g.assignedToSchedulingPod
	g.assignedToSchedulingPod = nil
	g.schedulingPodUID = ""
	g.put(assigned)
}

// unassignForBinding unassigns the guest instance from the pod in the binding cycle.
func (g *guestPool) unassignForBinding(podUID types.UID) {
	assigned := g.assignedToBindingPod[podUID]
	delete(g.assignedToBindingPod, podUID)
	g.put(assigned)
}

// put puts the guest instance back to the pool.
func (g *guestPool) put(guest *guest) {
	g.pool.Put(guest)
}

// getInstanceForSchedulingPod gets a guest instance for the Pod in the scheduling cycle.
// If the pod has already got a guest, it returns the guest.
// Otherwise, it gets a guest from the pool.
func (p *guestPool) getInstanceForSchedulingPod(ctx context.Context, podUID types.UID) (*guest, bool) {
	// Check if the pod has already got a guest.
	if podUID == p.schedulingPodUID {
		return p.assignedToSchedulingPod, true
	}

	// If not, get a guest from the pool.
	g := p.pool.Get()
	if g == nil {
		return nil, false
	}
	gue, ok := g.(*guest)
	if !ok {
		// shouldn't happen.
		return nil, false
	}

	return gue, true
}

var _ framework.FilterPlugin = (*wasmPlugin)(nil)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *wasmPlugin) Name() string {
	return PluginName
}

func (pl *wasmPlugin) getOrCreateGuest(ctx context.Context, podUID types.UID) (*guest, error) {
	poolG, ok := pl.pool.getInstanceForSchedulingPod(ctx, podUID)
	if !ok {
		if g, createErr := pl.newGuest(ctx); createErr != nil {
			return nil, createErr
		} else {
			poolG = g
			// Assign this new guest to the pod.
			pl.pool.assignForScheduling(poolG, podUID)
		}
	}

	// Assign the guest to the pod.
	pl.pool.assignForScheduling(poolG, podUID)

	return poolG, nil
}

func (pl *wasmPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (pl *wasmPlugin) PreFilter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	// When PreFilter is called, run unassign because the previous scheduling cycle should have been finished.
	pl.pool.unassignForScheduling()

	_, err := pl.getOrCreateGuest(ctx, pod.GetUID())
	if err != nil {
		return nil, framework.AsStatus(err)
	}

	// TODO: support PreFilter in wasm guest.

	return nil, nil
}

// Filter invoked at the filter extension point.
func (pl *wasmPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	g, err := pl.getOrCreateGuest(ctx, pod.GetUID())
	if err != nil {
		return framework.AsStatus(err)
	}

	// The guest Wasm may call host functions, so we add context parameters of
	// the current args.
	args := &filterArgs{pod: pod, nodeInfo: nodeInfo}
	ctx = context.WithValue(ctx, filterArgsKey{}, args)
	return g.filter(ctx)
}

func (pl *wasmPlugin) AddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	// TODO: support AddPod in wasm guest.
	return nil
}

func (pl *wasmPlugin) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	pl.pool.assignedToSchedulingPodLock.Lock()
	defer pl.pool.assignedToSchedulingPodLock.Unlock()

	// TODO: support RemovePod in wasm guest.
	return nil
}

func (pl *wasmPlugin) Permit(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	// assume that the pod is going to binding cycle and continue to assign the instance to the pod.
	// unassign the instance in Unreserve or PostBind.
	pl.pool.assignForBinding(p.GetUID())

	// TODO: support Permit in wasm guest.

	return nil, 0
}

func (pl *wasmPlugin) Reserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) *framework.Status {
	// TODO: support Reserve in wasm guest.
	// Currently, it's implemented to implement the ReservePlugin interface.
	return nil
}

func (pl *wasmPlugin) Unreserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	pl.pool.unassignForBinding(p.GetUID())
	// TODO: support Unreserve in wasm guest.
}

func (pl *wasmPlugin) PostBind(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	pl.pool.unassignForBinding(p.GetUID())
	// TODO: support PostBind in wasm guest.
}

// Close implements io.Closer
func (pl *wasmPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}
