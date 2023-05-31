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
)

// Filter should be assigned in `main` to a FilterFunc function.
//
// For example:
//
//	func main() {
//		guest.Filter = api.FilterFunc(nameEqualsPodSpec)
//	}
var Filter api.Filter

// filter is only exported to the host.
//
//go:export filter
func filter() (code uint32) { //nolint
	if Filter == nil {
		return
	}
	c, reason := Filter.Filter(filterArgs{})
	if reason != "" {
		imports.StatusReason(reason)
	}
	return uint32(c)
}

var _ api.FilterArgs = filterArgs{}

type filterArgs struct{}

func (filterArgs) NodeInfo() api.NodeInfo {
	return nodeInfo{}
}

func (filterArgs) Pod() *protoapi.Pod {
	b := imports.Pod()
	var msg protoapi.Pod
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err.Error())
	}
	return &msg
}

var _ api.NodeInfo = nodeInfo{}

type nodeInfo struct{}

func (nodeInfo) Node() *protoapi.Node {
	b := imports.NodeInfoNode()
	var msg protoapi.Node
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err)
	}
	return &msg
}
