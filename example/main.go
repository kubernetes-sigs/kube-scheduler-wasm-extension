package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

func main() {
	guest.Filter = api.FilterFunc(nameEqualsPodSpec)
}

// nameEqualsPodSpec schedules this node if its name equals its pod spec.
func nameEqualsPodSpec(args api.FilterArgs) (api.StatusCode, string) {
	nodeName := nilToEmpty(args.NodeInfo().Node().Metadata.Name)
	podSpecNodeName := nilToEmpty(args.Pod().Spec.NodeName)

	if len(podSpecNodeName) == 0 || podSpecNodeName == nodeName {
		return api.StatusCodeSuccess, ""
	} else {
		return api.StatusCodeUnschedulable, podSpecNodeName + " != " + nodeName
	}
}

func nilToEmpty(ptr *string) (s string) {
	if ptr != nil {
		s = *ptr
	}
	return
}
