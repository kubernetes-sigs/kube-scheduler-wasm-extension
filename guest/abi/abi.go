package abi

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/abi/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// Filter defaults to return success
var Filter api.Filter

// filter is only exported to the host.
//
//go:export filter
func filter() (code uint32) { //nolint
	if Filter == nil {
		return
	}
	c, reason := Filter.Filter(nodeInfo{}, pod{})
	if reason != "" {
		imports.Reason(reason)
	}
	return uint32(c)
}

var _ api.NodeInfo = nodeInfo{}

type nodeInfo struct{}

func (nodeInfo) Node() api.Node {
	return nodeInfo{}
}

func (nodeInfo) Name() string {
	return imports.NodeInfoNodeName()
}

var _ api.Pod = pod{}

type pod struct{}

func (pod) Spec() *protoapi.IoK8SApiCoreV1PodSpec {
	b := imports.PodSpec()
	var msg protoapi.IoK8SApiCoreV1PodSpec
	if err := msg.UnmarshalVT(b); err != nil {
		panic(err)
	}
	return &msg
}
