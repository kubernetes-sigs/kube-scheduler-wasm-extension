package api

import (
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

type Filter interface {
	Filter(NodeInfo, Pod) (code Code, reason string)
}

type NodeInfo interface {
	Node() Node
}

type Node interface {
	Name() string
}

type Pod interface {
	Spec() *protoapi.IoK8SApiCoreV1PodSpec
}
