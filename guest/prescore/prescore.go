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

// Package prescore exports an api.PreScorePlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package prescore

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// prescore is the current plugin assigned with SetPlugin.
var prescore api.PreScorePlugin

// SetPlugin should be called in `main` to assign an api.PreScorePlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := scorePlugin{}
//		prescore.SetPlugin(func(klog klogapi.Klog, jsonConfig []byte, h handleapi.Handle) (api.Plugin, error) { return plugin, nil })
//		score.SetPlugin(func(klog klogapi.Klog, jsonConfig []byte, h handleapi.Handle) (api.Plugin, error) { return plugin, nil })
//	}
//
//	type scorePlugin struct{}
//
//	func (scorePlugin) PreScore(state api.CycleState, pod proto.Pod, nodeList proto.NodeList) *api.Status {
//		// Write state you need on Score
//		return nil
//	}
//
//	func (scorePlugin) Score(state api.CycleState, pod proto.Pod, nodeName string) (int32, *api.Status) {
//		var score int32
//		// Derive score for the node name using state set on PreScore!
//		return score, nil
//	}
//
// Note: This should only be set when score.SetPlugin also is.
func SetPlugin(pluginFactory handleapi.PluginFactory) {
	handle := handle.NewFrameworkHandle()
	p, err := pluginFactory(klog.Get(), config.Get(), handle)
	if err != nil {
		panic(err)
	}
	var ok bool
	prescore, ok = p.(api.PreScorePlugin)
	if !ok || prescore == nil {
		panic("nil PreScorePlugin or a plugin is not compatible with PreScorePlugin type")
	}
	plugin.MustSet(prescore)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _prescore

// prescore is only exported to the host.
//
//export prescore
func _prescore() uint32 {
	if prescore == nil { // Then, the user didn't define one.
		// This is likely caused by use of plugin.Set(p), where 'p' didn't
		// implement PreScorePlugin: return success.
		return 0
	}

	// Pod is lazy and the same value for all plugins in a scheduling cycle.
	pod := cyclestate.Pod

	s := prescore.PreScore(cyclestate.Values, pod, &nodeList{})

	return imports.StatusToCode(s)
}

// nodeList is lazy so that a plugin which doesn't read fields avoids a
// relatively expensive unmarshal penalty.
//
// Note: Unlike proto.Pod, this is not special cased for the scheduling cycle.
type nodeList struct {
	items []proto.Node
}

func (n *nodeList) Items() []proto.Node {
	return n.lazyItems()
}

// lazyItems lazy initializes the nodes from lodeList.
func (n *nodeList) lazyItems() []proto.Node {
	if items := n.items; items != nil {
		return items
	}

	var msg protoapi.NodeList
	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	if err := mem.Update(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return k8sApiNodeList(ptr, limit)
	}, msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}

	size := len(msg.Items)
	if size == 0 {
		return nil
	}

	items := make([]proto.Node, size)
	for i := range msg.Items {
		items[i] = &internalproto.Node{Msg: msg.Items[i]}
	}
	n.items = items
	return items
}
