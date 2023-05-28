package guest

import (
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
