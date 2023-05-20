package vfs

import (
	v1 "k8s.io/api/core/v1"
	"os"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

func Filter(filter api.Filter) {
	// verify we support the args
	switch os.Args[1] {
	case "filter":
	default:
		panic(os.Args)
	}

	// The user can't recover any error fetching the nodeInfo or pod data.
	// Instead of having each function return error, panic and recover here.
	defer func() {
		if recovered := recover(); recovered != nil {
			if err, ok := recovered.(error); ok {
				println(err.Error()) // reason
				os.Exit(int(api.Error))
			} else {
				os.Exit(int(api.Error))
			}
		}
	}()

	if code, err := filter.Filter(nodeInfo{}, pod{}); err != nil {
		println(err.Error()) // reason
		os.Exit(int(api.Error))
	} else {
		os.Exit(int(code))
	}
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

func (pod) Spec() *v1.PodSpec {
	if b, err := os.ReadFile("/kdev/pod/spec"); err != nil {
		panic(err)
	} else {
		var ps v1.PodSpec
		if err = ps.Unmarshal(b); err != nil {
			panic(err)
		}
		return &ps
	}
}
