package e2e

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Fatalf func(format string, args ...any)

func RunAll(ctx context.Context, fatalf Fatalf, plugin framework.Plugin, pod *v1.Pod, ni *framework.NodeInfo) (score int64) {
	MaybeRunPreFilter(ctx, fatalf, plugin, pod)

	var s *framework.Status
	if filterP, ok := plugin.(framework.FilterPlugin); ok {
		s = filterP.Filter(ctx, nil, pod, ni)
		RequireSuccess(fatalf, s)
	}
	if scoreP, ok := plugin.(framework.ScorePlugin); ok {
		score, s = scoreP.Score(ctx, nil, pod, ni.Node().Name)
		RequireSuccess(fatalf, s)
	}
	return
}

// MaybeRunPreFilter calls framework.PreFilterPlugin, if defined, as that
// resets the cycle state.
func MaybeRunPreFilter(ctx context.Context, fatalf Fatalf, plugin framework.Plugin, pod *v1.Pod) {
	// We always implement EnqueueExtensions for simplicity
	_ = plugin.(framework.EnqueueExtensions).EventsToRegister()

	if p, ok := plugin.(framework.PreFilterPlugin); ok {
		_, s := p.PreFilter(ctx, nil, pod)
		RequireSuccess(fatalf, s)
	}
}

func RequireSuccess(fatalf Fatalf, s *framework.Status) {
	if want, have := framework.Success, s.Code(); want != have {
		fatalf("unexpected status code: want %v, have %v, reason: %v", want, have, s.Message())
	}
}
