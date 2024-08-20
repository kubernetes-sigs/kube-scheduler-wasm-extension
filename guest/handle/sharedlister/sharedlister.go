package sharedlister

import (
	guestapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister/internal"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/prefilter"
)

var sharedListerInstance api.SharedLister = &internal.SharedLister{
	NodeInfoList: prefilter.Nodes,
}

func Get() api.SharedLister {
	return sharedListerInstance
}

// NodeInfos is a convenience that calls the same method documented on api.NodeInfos.
func NodeInfos() guestapi.NodeInfoList {
	return sharedListerInstance.NodeInfos()
}
