package api

import (
	v1 "k8s.io/api/core/v1"
)

type Filter interface {
	Filter(NodeInfo, Pod) (Code, error)
}

type NodeInfo interface {
	Node() Node
}

type Node interface {
	Name() string
}

type Pod interface {
	Spec() *v1.PodSpec
}
