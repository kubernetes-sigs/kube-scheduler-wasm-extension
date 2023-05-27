package api

import protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"

// Filter is a WebAssembly implementation of framework.FilterPlugin.
type Filter interface {
	Filter(FilterArgs) (statusCode StatusCode, statusReason string)
}

// FilterFunc adapts an ordinary function to a Filter.
type FilterFunc func(FilterArgs) (statusCode StatusCode, statusReason string)

// Filter returns f(a).
func (f FilterFunc) Filter(a FilterArgs) (statusCode StatusCode, statusReason string) {
	return f(a)
}

// FilterArgs are the arguments to a Filter.
//
// Note: The arguments are lazy fetched to avoid overhead for properties not in use.
type FilterArgs interface {
	NodeInfo() NodeInfo
	Pod() *protoapi.Pod
}

type NodeInfo interface {
	Node() *protoapi.Node
}
