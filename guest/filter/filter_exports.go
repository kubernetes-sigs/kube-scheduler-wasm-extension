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

package filter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = filter

var plugin api.FilterPlugin

// filter is only exported to the host.
//
//export filter
func filter() uint32 { //nolint
	if plugin == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("filter imported, but filter.SetPlugin not called")
	}

	s := plugin.Filter(cyclestate.Pod, &nodeInfo{})

	return imports.StatusToCode(s)
}

var _ api.NodeInfo = (*nodeInfo)(nil)

// nodeInfo is lazy so that a plugin which doesn't read fields avoids a
// relatively expensive unmarshal penalty.
type nodeInfo struct {
	n *protoapi.Node
}

func (n *nodeInfo) Node() *protoapi.Node {
	return n.node()
}

func (n *nodeInfo) node() *protoapi.Node {
	if node := n.n; node != nil {
		return node
	}

	var msg protoapi.Node
	if err := imports.NodeInfoNode(msg.UnmarshalVT); err != nil {
		panic(err)
	}
	n.n = &msg
	return n.n
}
