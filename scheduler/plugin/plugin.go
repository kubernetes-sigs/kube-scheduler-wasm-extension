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
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

func PluginFactory(pluginName string) frameworkruntime.PluginFactory {
	return func(configuration runtime.Object, frameworkHandle framework.Handle) (framework.Plugin, error) {
		config := PluginConfig{}
		if err := frameworkruntime.DecodeInto(configuration, &config); err != nil {
			return nil, fmt.Errorf("failed to decode into %s PluginConfig: %w", pluginName, err)
		}
		return NewFromConfig(context.Background(), pluginName, config)
	}
}

// NewFromConfig is like New, except it allows us to explicitly provide the
// context and configuration of the plugin. This allows flexibility in tests.
func NewFromConfig(ctx context.Context, pluginName string, config PluginConfig) (framework.Plugin, error) {
	url := config.GuestURL
	if url == "" {
		return nil, errors.New("wasm: guestURL is required")
	}
	guestBin, err := getURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("wasm: error reading guestURL %s: %w", url, err)
	}

	runtime, guestModule, err := prepareRuntime(ctx, guestBin, config.LogSeverity, config.GuestConfig)
	if err != nil {
		return nil, err
	}

	pl, err := newWasmPlugin(ctx, pluginName, runtime, guestModule, config)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, err
	}

	// The scheduler framework uses type assertions, so mask based on what
	// the guest exports.
	if pl, err := maskInterfaces(pl); err != nil {
		_ = runtime.Close(ctx)
		return nil, err
	} else {
		return pl, nil
	}
}

// newWasmPlugin is extracted to prevent small bugs: The caller must close the
// wazero.Runtime to avoid leaking mmapped files.
func newWasmPlugin(ctx context.Context, pluginName string, runtime wazero.Runtime, guestModule wazero.CompiledModule, config PluginConfig) (*wasmPlugin, error) {
	var guestInterfaces interfaces
	var err error
	if guestInterfaces, err = detectInterfaces(guestModule.ExportedFunctions()); err != nil {
		return nil, err
	} else if guestInterfaces == 0 {
		return nil, fmt.Errorf("wasm: guest doesn't export plugin functions")
	}

	pl := &wasmPlugin{
		pluginName:        pluginName,
		runtime:           runtime,
		guestModule:       guestModule,
		guestArgs:         config.Args,
		guestInterfaces:   guestInterfaces,
		guestModuleConfig: wazero.NewModuleConfig(),
		instanceCounter:   atomic.Uint64{},
	}

	if pl.pool, err = newGuestPool(ctx, pl.newGuest); err != nil {
		return nil, fmt.Errorf("failed to create a guest pool: %w", err)
	}
	return pl, nil
}

type wasmPlugin struct {
	pluginName        string
	runtime           wazero.Runtime
	guestModule       wazero.CompiledModule
	guestInterfaces   interfaces
	guestModuleConfig wazero.ModuleConfig
	instanceCounter   atomic.Uint64
	pool              *guestPool[*guest]
	guestArgs         []string
}

// ProfilerSupport exposes functions needed to profile the guest with wzprof.
type ProfilerSupport interface {
	Guest() wazero.CompiledModule
	plugin() *wasmPlugin
}

func (pl *wasmPlugin) Guest() wazero.CompiledModule {
	return pl.guestModule
}

func (pl *wasmPlugin) plugin() *wasmPlugin {
	return pl
}

var _ framework.Plugin = (*wasmPlugin)(nil)

// Name implements the same method as documented on framework.Plugin.
// The plugin name is defined by the scheduler configuration.
func (pl *wasmPlugin) Name() string {
	return pl.pluginName
}

var _ framework.EnqueueExtensions = (*wasmPlugin)(nil)

// allClusterEvents is copied from framework.go, to avoid the complexity of
// conditionally implementing framework.EnqueueExtensions.
var allClusterEvents = []framework.ClusterEvent{
	{Resource: framework.Pod, ActionType: framework.All},
	{Resource: framework.Node, ActionType: framework.All},
	{Resource: framework.CSINode, ActionType: framework.All},
	{Resource: framework.PersistentVolume, ActionType: framework.All},
	{Resource: framework.PersistentVolumeClaim, ActionType: framework.All},
	{Resource: framework.StorageClass, ActionType: framework.All},
}

// EventsToRegister implements the same method as documented on framework.EnqueueExtensions.
func (pl *wasmPlugin) EventsToRegister() (clusterEvents []framework.ClusterEvent) {
	// We always implement EventsToRegister, even when the guest doesn't
	if pl.guestInterfaces&iEnqueueExtensions == 0 {
		return allClusterEvents // unimplemented
	}

	// TODO: Track https://github.com/kubernetes/kubernetes/pull/119155 for QueueingHintFn
	// This will become []ClusterEventWithHint, but the hint will be difficult
	// to implement. If we do, we may need to make a generic guest export
	// QueueingHintFn which has another parameter of the an funcID which
	// substitutes for event.QueueingHintFn and passed later to the generic
	// guest export. There will be other concerns as the parameters to
	// QueueingHintFn are `interface{}` so probably need to be narrowed to
	// support at all.

	// Note: EventsToRegister() doesn't pass a context or return an error
	// See https://github.com/kubernetes/kubernetes/issues/119323/
	ctx := context.Background()

	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{}
	ctx = context.WithValue(ctx, stackKey{}, params)
	clusterEvents = allClusterEvents // On any problem fallback to default

	// Enqueue is not a part of the scheduling cycle.
	// Note: there's no error return from EventsToRegister()
	if err := pl.pool.doWithGuest(ctx, func(g *guest) {
		// Only override the default cluster events if at least one was
		// returned from the guest
		if ce := g.eventsToRegister(ctx); len(ce) != 0 {
			clusterEvents = ce
		}
	}); err != nil {
		panic(err)
	}
	return
}

var _ framework.PreFilterExtensions = (*wasmPlugin)(nil)

// AddPod implements the same method as documented on framework.PreFilterExtensions.
func (pl *wasmPlugin) AddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	// We implement PreFilterExtensions with FilterPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreFilterExtensions == 0 {
		return nil // unimplemented
	}
	panic("TODO: scheduling: AddPod")
}

// RemovePod implements the same method as documented on framework.PreFilterExtensions.
func (pl *wasmPlugin) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	// We implement PreFilterExtensions with FilterPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreFilterExtensions == 0 {
		return nil // unimplemented
	}
	panic("TODO: scheduling: RemovePod")
}

var _ framework.PreFilterPlugin = (*wasmPlugin)(nil)

// PreFilterExtensions implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	// We implement PreFilterExtensions with FilterPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreFilterExtensions == 0 {
		return nil // unimplemented
	}
	return pl
}

// PreFilter implements the same method as documented on
// framework.PreFilterPlugin.
func (pl *wasmPlugin) PreFilter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod) (result *framework.PreFilterResult, status *framework.Status) {
	// We implement PreFilterPlugin with FilterPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreFilterPlugin == 0 {
		return nil, nil // unimplemented
	}

	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		var nodeNames []string
		nodeNames, status = g.preFilter(ctx)
		if nodeNames != nil {
			result = &framework.PreFilterResult{NodeNames: sets.NewString(nodeNames...)}
		}
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.FilterPlugin = (*wasmPlugin)(nil)

// Filter implements the same method as documented on framework.FilterPlugin.
func (pl *wasmPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) (status *framework.Status) {
	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, node: nodeInfo.Node()}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		status = g.filter(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.PostFilterPlugin = (*wasmPlugin)(nil)

// PostFilter implements the same method as documented on framework.PostFilterPlugin.
func (pl *wasmPlugin) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (result *framework.PostFilterResult, status *framework.Status) {
	// We implement PostFilterPlugin with FilterPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPostFilterPlugin == 0 {
		return nil, nil // unimplemented
	}

	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, nodeToStatusMap: filteredNodeStatusMap}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		result, status = g.postFilter(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.PreScorePlugin = (*wasmPlugin)(nil)

// PreScore implements the same method as documented on framework.PreScorePlugin.
func (pl *wasmPlugin) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) (status *framework.Status) {
	// We implement PreScorePlugin with ScorePlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreScorePlugin == 0 {
		return nil // unimplemented
	}

	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, nodes: nodes}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		status = g.preScore(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

var _ framework.ScoreExtensions = (*wasmPlugin)(nil)

// NormalizeScore implements the same method as documented on framework.ScoreExtensions.
func (pl *wasmPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) (status *framework.Status) {
	// We implement ScoreExtensions with ScorePlugin, even when the guest doesn't.
	if pl.guestInterfaces&iScoreExtensions == 0 {
		return nil // unimplemented
	}
	params := &stack{pod: pod, nodeScoreList: scores}
	ctx = context.WithValue(ctx, stackKey{}, params)
	var updatedScores framework.NodeScoreList
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		updatedScores, status = g.normalizeScore(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	if len(scores) != len(updatedScores) {
		panic(fmt.Sprintf("size mismatch: scores has %d elements, but updatedScores has %d elements", len(scores), len(updatedScores)))
	}
	// Copy the contents of updatedScores into scores, modifying scores by reference
	for i := range scores {
		scores[i] = updatedScores[i]
	}
	return
}

var _ framework.ScorePlugin = (*wasmPlugin)(nil)

// Score implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (score int64, status *framework.Status) {
	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		score, status = g.score(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

// ScoreExtensions implements the same method as documented on framework.ScorePlugin.
func (pl *wasmPlugin) ScoreExtensions() framework.ScoreExtensions {
	// We implement ScoreExtensions with ScorePlugin, even when the guest doesn't.
	if pl.guestInterfaces&iScoreExtensions == 0 {
		return nil // unimplemented
	}
	return pl
}

var _ framework.ReservePlugin = (*wasmPlugin)(nil)

// Reserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (status *framework.Status) {
	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		status = g.reserve(ctx)
	}); err != nil {
		status = framework.AsStatus(err)
	}
	return
}

// Unreserve implements the same method as documented on framework.ReservePlugin.
func (pl *wasmPlugin) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	defer pl.pool.freeFromBinding(pod.UID) // the cycle is over, put it back into the pool.

	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	logger := klog.FromContext(ctx)
	if err := pl.pool.doWithSchedulingGuest(ctx, pod.UID, func(g *guest) {
		g.unreserve(ctx)
	}); err != nil {
		logger.Error(err, "doWithSchedulingGuest Failed")
	}
}

var _ framework.PreBindPlugin = (*wasmPlugin)(nil)

// PreBind implements the same method as documented on framework.PreBindPlugin.
func (pl *wasmPlugin) PreBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (status *framework.Status) {
	// We implement PreBindPlugin with BindPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPreBindPlugin == 0 {
		return nil // unimplemented
	}

	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	g := pl.pool.getForBinding(pod.UID)
	status = g.preBind(ctx)
	return
}

var _ framework.PostBindPlugin = (*wasmPlugin)(nil)

// PostBind implements the same method as documented on framework.PostBindPlugin.
func (pl *wasmPlugin) PostBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	// We implement PostBindPlugin with BindPlugin, even when the guest doesn't.
	if pl.guestInterfaces&iPostBindPlugin == 0 {
		return // unimplemented
	}

	defer pl.pool.freeFromBinding(pod.UID) // the cycle is over, put it back into the pool.
	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	g := pl.pool.getForBinding(pod.UID)
	g.postBind(ctx)
}

var _ framework.PermitPlugin = (*wasmPlugin)(nil)

// Permit implements the same method as documented on framework.PermitPlugin.
func (pl *wasmPlugin) Permit(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	_ = pl.pool.getForBinding(pod.UID)

	// TODO: partially implemented for testing

	return nil, 0
}

var _ framework.BindPlugin = (*wasmPlugin)(nil)

// Bind implements the same method as documented on framework.BindPlugin.
func (pl *wasmPlugin) Bind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (status *framework.Status) {
	// Add the stack to the go context so that the corresponding host function
	// can look them up.
	params := &stack{pod: pod, nodeName: nodeName}
	ctx = context.WithValue(ctx, stackKey{}, params)
	g := pl.pool.getForBinding(pod.UID)
	status = g.bind(ctx)
	return
}

// Close implements io.Closer
func (pl *wasmPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}
