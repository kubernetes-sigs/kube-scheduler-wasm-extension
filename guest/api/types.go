package api

import protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"

// Filter is a WebAssembly implementation of framework.FilterPlugin.
type Filter interface {
	Filter(NodeInfo, Pod) (statusCode StatusCode, statusReason string)
}

// FilterFunc adapts an ordinary function to a Filter.
type FilterFunc func(NodeInfo, Pod) (statusCode StatusCode, statusReason string)

// Filter returns f(n, p).
func (f FilterFunc) Filter(n NodeInfo, p Pod) (statusCode StatusCode, statusReason string) {
	return f(n, p)
}

type NodeInfo interface {
	Node() *protoapi.IoK8SApiCoreV1Node
}

type Pod interface {
	Spec() *protoapi.IoK8SApiCoreV1PodSpec
}
