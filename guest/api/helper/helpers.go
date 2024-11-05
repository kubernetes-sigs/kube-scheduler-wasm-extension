package helper

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// TolerationsTolerateTaint checks if taint is tolerated by any of the tolerations.
func TolerationsTolerateTaint(tolerations []*protoapi.Toleration, taint *protoapi.Taint) bool {
	for i := range tolerations {
		if api.ToleratesTaint(tolerations[i], taint) {
			return true
		}
	}
	return false
}
