package api

import v1 "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"

const (
	// TaintNodeUnschedulable will be added when node becomes unschedulable
	// and removed when node becomes schedulable.
	TaintNodeUnschedulable string = "node.kubernetes.io/unschedulable"
	// Do not allow new pods to schedule onto the node unless they tolerate the taint,
	// but allow all pods submitted to Kubelet without going through the scheduler
	// to start, and allow all already-running pods to continue running.
	// Enforced by the scheduler.
	TaintEffectNoSchedule string = "NoSchedule"
	TolerationOpExists    string = "Exists"
	TolerationOpEqual     string = "Equal"
)

// ToleratesTaint checks if the toleration tolerates the taint.
// The matching follows the rules below:
//
//  1. Empty toleration.effect means to match all taint effects,
//     otherwise taint effect must equal to toleration.effect.
//  2. If toleration.operator is 'Exists', it means to match all taint values.
//  3. Empty toleration.key means to match all taint keys.
//     If toleration.key is empty, toleration.operator must be 'Exists';
//     this combination means to match all taint values and all taint keys.
func ToleratesTaint(toleration *v1.Toleration, taint *v1.Taint) bool {
	if len(toleration.GetEffect()) > 0 && toleration.Effect != taint.Effect {
		return false
	}

	if len(toleration.GetKey()) > 0 && toleration.Key != taint.Key {
		return false
	}

	// TODO: Use proper defaulting when Toleration becomes a field of PodSpec
	switch *toleration.Operator {
	// empty operator means Equal
	case "", TolerationOpEqual:
		return toleration.Value == taint.Value
	case TolerationOpExists:
		return true
	default:
		return false
	}
}
