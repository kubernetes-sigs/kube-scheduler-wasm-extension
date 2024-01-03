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

// Package enqueue is defined internally so that it can export Pod as
// cyclestate.Pod, without circular dependencies or exporting it publicly.
package enqueue

import (
	"runtime"
	"unsafe"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue/clusterevent"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
)

// enqueue is the current plugin assigned with SetPlugin.
var enqueue api.EnqueueExtensions

// SetPlugin is exposed to prevent package cycles.
func SetPlugin(pluginFactory handleapi.PluginFactory) {
	handle := handle.NewFrameworkHandle()
	p, err := pluginFactory(klog.Get(), config.Get(), handle)
	if err != nil {
		panic(err)
	}
	var ok bool
	enqueue, ok = p.(api.EnqueueExtensions)
	if !ok || enqueue == nil {
		panic("nil EnqueueExtensions or a plugin is not compatible with EnqueueExtensions type")
	}
	plugin.MustSet(enqueue)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() = _enqueue

// enqueue is only exported to the host.
//
//export enqueue
func _enqueue() {
	if enqueue == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement EnqueueExtensions: return to use default events.
		return
	}

	clusterEvents := enqueue.EventsToRegister()

	// If plugin returned clusterEvents, encode them and call the host with the
	// count and memory region.
	encoded := clusterevent.EncodeClusterEvents(clusterEvents)
	if encoded != nil {
		ptr := uint32(uintptr(unsafe.Pointer(&encoded[0])))
		size := uint32(len(encoded))
		setClusterEventsResult(ptr, size)
		runtime.KeepAlive(encoded) // until ptr is no longer needed.
	}
}
