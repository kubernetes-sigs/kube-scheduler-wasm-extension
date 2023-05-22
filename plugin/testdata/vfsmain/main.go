package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/vfs"
)

func main() {
	vfs.Filter(exampleFilter{})
}

var _ api.Filter = exampleFilter{}

type exampleFilter struct{}

func (exampleFilter) Filter(nodeInfo api.NodeInfo, pod api.Pod) (api.Code, string) {
	nodeName := nodeInfo.Node().Name()
	podSpecNodeName := pod.Spec().NodeName

	if len(podSpecNodeName) == 0 || podSpecNodeName == nodeName {
		return api.Success, ""
	} else {
		return api.Unschedulable, ""
	}
}
