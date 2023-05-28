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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type WasmPlugin struct{ *wasmPlugin }

func NewTestWasmPlugin(p framework.Plugin) (*WasmPlugin, bool) {
	pl, ok := p.(*wasmPlugin)
	if !ok {
		return nil, false
	}

	return &WasmPlugin{wasmPlugin: pl}, true
}

func (w *WasmPlugin) GetOrCreateGuest(ctx context.Context, podUID types.UID) (*guest, error) {
	return w.getOrCreateGuest(ctx, podUID)
}

func (w *WasmPlugin) ClearGuestModule() {
	w.guestModule = nil
}

func (w *WasmPlugin) GetSchedulingPodUID() types.UID {
	return w.pool.schedulingPodUID
}

func (w *WasmPlugin) GetAssignedToBindingPod() map[types.UID]*guest {
	return w.pool.assignedToBindingPod
}

func (w *WasmPlugin) GetInstanceFromPool() any {
	return w.pool.pool.Get()
}
