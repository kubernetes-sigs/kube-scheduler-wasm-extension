package api

import (
	guestapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

type SharedLister interface {
	NodeInfos() guestapi.NodeInfoList
}

type UnimplementedSharedLister struct{}

func (UnimplementedSharedLister) NodeInfos() guestapi.NodeInfoList {
	return nil
}
