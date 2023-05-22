package vfs

import (
	"os"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

func Filter(filter api.Filter) {
	// verify we support the args
	switch os.Args[1] {
	case "filter":
	default:
		panic(os.Args)
	}

	code, reason := filter.Filter(nodeInfo{}, pod{})
	if reason != "" {
		println(reason) // anything in STDERR becomes the reason
	}
	os.Exit(int(code))
}

var _ api.NodeInfo = nodeInfo{}

// TODO: proto serialization code compatible with v1.NodeInfo.Marshal
type nodeInfo struct{}

func (nodeInfo) Node() api.Node {
	return nodeInfo{}
}

func (nodeInfo) Name() string {
	if b, err := os.ReadFile("/kdev/nodeInfo/node/name"); err != nil {
		panic(err)
	} else {
		return string(b)
	}
}

var _ api.Pod = pod{}

type pod struct{}

func (pod) Spec() *protoapi.IoK8SApiCoreV1PodSpec {
	if b, err := os.ReadFile("/kdev/pod/spec"); err != nil {
		panic(err)
	} else {
		var msg protoapi.IoK8SApiCoreV1PodSpec
		if err := msg.UnmarshalVT(b); err != nil {
			panic(err)
		}
		return &msg
	}
}
