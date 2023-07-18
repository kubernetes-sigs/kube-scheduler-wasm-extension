package e2e

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Testing interface {
	Fatalf(format string, args ...any)
	Helper()
}

func RunAll(ctx context.Context, t Testing, plugin framework.Plugin, pod *v1.Pod, ni *framework.NodeInfo) (score int64) {
	t.Helper()

	MaybeRunPreFilter(ctx, t, plugin, pod)

	var s *framework.Status
	if filterP, ok := plugin.(framework.FilterPlugin); ok {
		s = filterP.Filter(ctx, nil, pod, ni)
		RequireSuccess(t, s)
	}

	if prescoreP, ok := plugin.(framework.PreScorePlugin); ok {
		s = prescoreP.PreScore(ctx, nil, pod, []*v1.Node{ni.Node()})
		RequireSuccess(t, s)
	}

	if scoreP, ok := plugin.(framework.ScorePlugin); ok {
		score, s = scoreP.Score(ctx, nil, pod, ni.Node().Name)
		RequireSuccess(t, s)
	}
	return
}

// MaybeRunPreFilter calls framework.PreFilterPlugin, if defined, as that
// resets the cycle state.
func MaybeRunPreFilter(ctx context.Context, t Testing, plugin framework.Plugin, pod *v1.Pod) {
	t.Helper()

	// We always implement EnqueueExtensions for simplicity
	_ = plugin.(framework.EnqueueExtensions).EventsToRegister()

	if p, ok := plugin.(framework.PreFilterPlugin); ok {
		_, s := p.PreFilter(ctx, nil, pod)
		RequireSuccess(t, s)
	}
}

func RequireSuccess(t Testing, s *framework.Status) {
	t.Helper()

	if want, have := framework.Success, s.Code(); want != have {
		t.Fatalf("unexpected status code: want %v, have %v, reason: %v", want, have, s.Message())
	}
}
