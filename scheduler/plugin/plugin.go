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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	ctx := context.Background()

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
