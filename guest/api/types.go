package api

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
	Spec() PodSpec
}

type PodSpec interface {
	NodeName() string
}
