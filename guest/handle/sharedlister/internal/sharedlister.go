package internal

import guestapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"

type SharedLister struct {
	NodeInfoList guestapi.NodeInfoList
}

func (s SharedLister) NodeInfos() guestapi.NodeInfoList {
	return s.NodeInfoList
}
