/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package e2e_test

import (
	"context"
	"io"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

// TestGuest_CycleStateCoherence ensures cycle state data is coherent in a
// scheduling context.
func TestGuest_CycleStateCoherence(t *testing.T) {
	ctx := context.Background()

	plugin, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: test.PathTestCycleState})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer plugin.(io.Closer).Close()

	pod := test.PodReal
	ni := framework.NewNodeInfo()
	ni.SetNode(test.NodeReal)

	// run: the guest will crash if any of the callbacks see a different pod.
	runAll(ctx, t, plugin, pod, ni)
	// run again: the guest will crash if it sees the same pointer.
	runAll(ctx, t, plugin, pod, ni)
}

// maybeRunPreFilter calls framework.PreFilterPlugin, if defined, as that
// resets the cycle state.
func maybeRunPreFilter[c common](ctx context.Context, t c, plugin framework.Plugin, pod *v1.Pod) {
	// We always implement EnqueueExtensions for simplicity
	_ = plugin.(framework.EnqueueExtensions).EventsToRegister()

	if p, ok := plugin.(framework.PreFilterPlugin); ok {
		_, s := p.PreFilter(ctx, nil, pod)
		requireSuccess(t, s)
	}
}

func runAll[c common](ctx context.Context, t c, plugin framework.Plugin, pod *v1.Pod, ni *framework.NodeInfo) {
	maybeRunPreFilter(ctx, t, plugin, pod)

	if f, ok := plugin.(framework.FilterPlugin); ok {
		s := f.Filter(ctx, nil, pod, ni)
		requireSuccess(t, s)
	}
	if s, ok := plugin.(framework.ScorePlugin); ok {
		_, s := s.Score(ctx, nil, pod, ni.Node().Name)
		requireSuccess(t, s)
	}
}

func requireSuccess[c common](t c, s *framework.Status) {
	if want, have := framework.Success, s.Code(); want != have {
		t.Fatalf("unexpected status code: want %v, have %v, reason: %v", want, have, s.Message())
	}
}

type common interface {
	Fatalf(format string, args ...any)
}
