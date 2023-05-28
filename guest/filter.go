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

import (
	"context"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/framework"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// filter should be assigned in `main` to a FilterFunc function.
//
// For example:
//
//	func main() {
//		guest.filter = api.FilterFunc(nameEqualsPodSpec)
//	}
var filter api.Filter

func RegisterFilter(pl framework.FilterPlugin) {
	filter = api.FilterFunc(func(fa api.FilterArgs) (statusCode framework.Code, statusReason string) {
		ctx := context.Background()
		status := pl.Filter(ctx, &framework.CycleState{}, fa.Pod, fa.NodeInfo)
		var reason string
		if len(status.Reasons()) > 0 {
			reason = status.Reasons()[0]
		}
		return status.Code(), reason
	})
}

// _filter is only exported to the host.
//
//go:export filter
func _filter() (code uint32) { //nolint
	if filter == nil {
		return
	}
	c, reason := filter.Filter(filterArgs{})
	if reason != "" {
		imports.StatusReason(reason)
	}
	return uint32(c)
}

var _ api.FilterArgs = filterArgs{}

type filterArgs struct{}

func (filterArgs) NodeInfo() framework.NodeInfo {
	return &nodeInfo{}
}

func (filterArgs) Pod() *protoapi.Pod {
	b := imports.Pod()
	var msg protoapi.Pod
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err.Error())
	}
	return &msg
}

var _ framework.NodeInfo = nodeInfo{}

type nodeInfo struct{}

func (nodeInfo) Node() *protoapi.Node {
	b := imports.NodeInfoNode()
	var msg protoapi.Node
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err)
	}
	return &msg
}
