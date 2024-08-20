// Package cyclestate holds fields scoped to the Pod Scheduling context, e.g.
// the beginning of the scheduling cycle until the end of the binding cycle.
//
// This field is reset on prefilter.Plugin, as that begins a new scheduling
// cycle. Even if the pod is the same as a failed cycle, its state must be
// reset so that any change is visible to the guest plugin.
//
// See https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#extension-points
package cyclestate

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/prefilter"
)

// Pod is the current pod being scheduled. It is lazy and the same values are
// returned for any plugins in a scheduling cycle.
var Pod proto.Pod = prefilter.CurrentPod

var Values api.CycleState = prefilter.CycleState
