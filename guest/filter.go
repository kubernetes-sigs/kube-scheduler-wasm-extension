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

package guest

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// FilterPlugin should be assigned in `main` to a FilterPlugin instance.
//
// For example:
//
//	func main() {
//		guest.FilterPlugin = api.FilterFunc(nameEqualsPodSpec)
//	}
var FilterPlugin api.FilterPlugin

// filter is only exported to the host.
//
//go:export filter
func filter() uint32 { //nolint
	// Pass on unconfigured filter
	if FilterPlugin == nil {
		return uint32(api.StatusCodeSuccess)
	}

	// The parameters passed are lazy with regard to host functions. This means
	// a no-op plugin should not have any unmarshal penalty.
	// TODO: Make these fields and reset on pre-filter or similar.
	s := FilterPlugin.Filter(&nodeInfo{}, &pod{})
	return imports.StatusToCode(s)
}

var _ api.NodeInfo = (*nodeInfo)(nil)

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

	b := imports.NodeInfoNode()
	var msg protoapi.Node
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err)
	}
	n.n = &msg
	return n.n
}

var _ api.Pod = (*pod)(nil)

type pod struct {
	p *protoapi.Pod
}

func (p *pod) Metadata() *meta.ObjectMeta {
	return p.pod().Metadata
}

func (p *pod) Spec() *protoapi.PodSpec {
	return p.pod().Spec
}

func (p *pod) Status() *protoapi.PodStatus {
	return p.pod().Status
}

// pod lazy initializes p from the imported host function imports.Pod.
func (p *pod) pod() *protoapi.Pod {
	if pod := p.p; pod != nil {
		return pod
	}

	b := imports.Pod()
	var msg protoapi.Pod
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err.Error())
	}
	p.p = &msg
	return p.p
}
