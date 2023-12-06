package prefilter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// Pod is exposed for the cyclestate package.
var Pod proto.Pod = pod{}

// CycleState is exposed for the cyclestate package.
var CycleState api.CycleState = cycleState{}

var currentCycleState = map[string]any{}

type cycleState struct{}

func (cycleState) Read(key string) (val any, ok bool) {
	val, ok = currentCycleState[key]
	return
}

func (cycleState) Write(key string, val any) {
	currentCycleState[key] = val
}

func (cycleState) Delete(key string) {
	delete(currentCycleState, key)
}

type pod struct{}

func (pod) GetName() string {
	return internalproto.GetName(lazyPod())
}

func (pod) GetNamespace() string {
	return internalproto.GetNamespace(lazyPod())
}

func (pod) GetUid() string {
	return internalproto.GetUid(lazyPod())
}

func (pod) GetResourceVersion() string {
	return internalproto.GetResourceVersion(lazyPod())
}

func (pod) GetKind() string {
	return "Pod"
}

func (pod) GetApiVersion() string {
	return "v1"
}

func (pod) Spec() *protoapi.PodSpec {
	return lazyPod().Spec
}

func (pod) Status() *protoapi.PodStatus {
	return lazyPod().Status
}

var currentPod *protoapi.Pod

// lazyPod lazy initializes currentPod from imports.Pod.
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
