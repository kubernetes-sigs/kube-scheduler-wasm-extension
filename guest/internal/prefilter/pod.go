package prefilter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// Pod is exposed for the cyclestate package.
var Pod api.Pod = pod{}

type pod struct{}

func (pod) Metadata() *meta.ObjectMeta {
	return lazyPod().Metadata
}

func (pod) Spec() *protoapi.PodSpec {
	return lazyPod().Spec
}

func (pod) Status() *protoapi.PodStatus {
	return lazyPod().Status
}

var currentPod *protoapi.Pod

// Pod lazy initializes p from the imported host function imports.Pod.
func lazyPod() *protoapi.Pod {
	if pod := currentPod; pod != nil {
		return pod
	}

	var msg protoapi.Pod
	if err := imports.Pod(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	currentPod = &msg
	return currentPod
}
