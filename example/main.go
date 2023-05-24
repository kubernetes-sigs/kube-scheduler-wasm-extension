package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

func main() {
	guest.Filter = api.FilterFunc(nameEqualsPosSpec)
}

// nameEqualsPosSpec schedules this node if its name equals its pod spec.
func nameEqualsPosSpec(nodeInfo api.NodeInfo, pod api.Pod) (api.StatusCode, string) {
	nodeName := nodeInfo.Node().Metadata.Name
	podSpecNodeName := pod.Spec().NodeName

	if len(podSpecNodeName) == 0 || podSpecNodeName == nodeName {
		return api.StatusCodeSuccess, ""
	} else {
		return api.StatusCodeUnschedulable, ""
	}
}
