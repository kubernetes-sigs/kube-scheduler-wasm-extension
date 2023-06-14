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

package wasm_test

import (
	"context"
	"fmt"
	"io"
	"math"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	st "k8s.io/kubernetes/pkg/scheduler/testing"

	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

var ctx = context.Background()

// Test_guestPool_assignedToBindingPod tests that the assignedToBindingPod field is set correctly.
func Test_guestPool_assignedToBindingPod(t *testing.T) {
	p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: test.PathExampleFilterSimple})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.Close()

	pl := wasm.NewTestWasmPlugin(p)
	pod := st.MakePod().UID("uid1").Name("test-pod").Node("good-node").Obj()
	nextPod := st.MakePod().UID("uid2").Name("test-pod2").Node("good-node").Obj()

	_, status := pl.PreFilter(ctx, nil, pod)
	if !status.IsSuccess() {
		t.Fatalf("prefilter failed: %v", status)
	}

	if pl.GetSchedulingPodUID() != pod.UID {
		t.Fatalf("expected schedulingPodUID to be %v, got %v", pod.UID, pl.GetSchedulingPodUID())
	}

	// pod is going to the binding cycle.
	status, _ = pl.Permit(ctx, nil, pod, "node")
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status)
	}

	if len(pl.GetAssignedToBindingPod()) != 1 {
		t.Fatalf("expected assignedToBindingPod to have 1 entry for `pod`, got %v", len(pl.GetAssignedToBindingPod()))
	}

	// another scheduling cycle for nextPod is started.

	_, status = pl.PreFilter(ctx, nil, nextPod)
	if !status.IsSuccess() {
		t.Fatalf("prefilter failed: %v", status)
	}

	if pl.GetSchedulingPodUID() != nextPod.UID {
		t.Fatalf("expected schedulingPodUID to be %v, got %v", pod.UID, pl.GetSchedulingPodUID())
	}

	status, _ = pl.Permit(ctx, nil, nextPod, "node")
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status)
	}

	if len(pl.GetAssignedToBindingPod()) != 2 {
		t.Fatalf("expected assignedToBindingPod to have 2 entry for `pod`, got %v", len(pl.GetAssignedToBindingPod()))
	}

	// make sure that the assignedToBindingPod has entries for both `pod` and `nextPod`.

	registeredPodUIDs := sets.New[types.UID]()
	for podUID := range pl.GetAssignedToBindingPod() {
		registeredPodUIDs.Insert(podUID)
	}
	if !registeredPodUIDs.Has(pod.UID) || !registeredPodUIDs.Has(nextPod.UID) {
		t.Fatalf("expected assignedToBindingPod to have entries for `pod` and `nextPod`, but got %v", registeredPodUIDs)
	}

	// pod is rejected in the binding cycle.
	pl.Unreserve(ctx, nil, pod, "node")
	if len(pl.GetAssignedToBindingPod()) != 1 {
		t.Fatalf("expected assignedToBindingPod to have 1 entry for `nextPod`, got %v", len(pl.GetAssignedToBindingPod()))
	}
	if _, ok := pl.GetAssignedToBindingPod()[nextPod.UID]; !ok {
		t.Fatalf("expected assignedToBindingPod to have entry for `nextPod`, got %v", pl.GetAssignedToBindingPod())
	}

	// nextPod is rejected in the binding cycle.
	pl.PostBind(ctx, nil, nextPod, "node")
	if len(pl.GetAssignedToBindingPod()) != 0 {
		t.Fatalf("expected assignedToBindingPod to have 0 entry, got %v", len(pl.GetAssignedToBindingPod()))
	}
}

// Test_guestPool_assignedToSchedulingPod tests that the schedulingPodUID is assigned during PreFilter expectedly.
func Test_guestPool_assignedToSchedulingPod(t *testing.T) {
	p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: test.PathExampleFilterSimple})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.Close()

	pl := wasm.NewTestWasmPlugin(p)
	pod := st.MakePod().UID("uid1").Name("test-pod").Node("good-node").Obj()
	nextPod := st.MakePod().UID("uid2").Name("test-pod2").Node("good-node").Obj()

	_, status := pl.PreFilter(ctx, nil, pod)
	if !status.IsSuccess() {
		t.Fatalf("prefilter failed: %v", status)
	}

	node := st.MakeNode().Name("good-node").Obj()
	ni := framework.NewNodeInfo()
	ni.SetNode(node)

	status = pl.Filter(ctx, nil, pod, ni)
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status)
	}

	if pl.GetSchedulingPodUID() != pod.UID {
		t.Fatalf("expected schedulingPodUID to be %v, got %v", pod.UID, pl.GetSchedulingPodUID())
	}

	// PreFilter is called with a different pod, meaning the past scheduling cycle of `pod` is finished.
	pl.PreFilter(ctx, nil, nextPod)

	if pl.GetSchedulingPodUID() != nextPod.UID {
		t.Fatalf("expected schedulingPodUID to be %v, got %v", nextPod.UID, pl.GetSchedulingPodUID())
	}

	if pl.GetInstanceFromPool() == nil {
		t.Fatal("expected guest instance that is used for `pod` to be in the pool, but it's not")
	}
}

// TestNew_masksInterfaces ensures the type returned by New can be asserted
// against, based on the statusCode in the guest.
func TestNew_masksInterfaces(t *testing.T) {
	tests := []struct {
		name         string
		guestPath    string
		expectFilter bool
		expectScore  bool
		expectBind   bool // currently a mask test until we implement bind
	}{
		{
			name:         "filter",
			guestPath:    test.PathExampleFilterSimple,
			expectFilter: true,
		},
		{
			name:        "score",
			guestPath:   test.PathExampleScoreSimple,
			expectScore: true,
		},
		{
			name:         "filter|score",
			guestPath:    test.PathTestAllNoopWat,
			expectFilter: true,
			expectScore:  true,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := wasm.New(&runtime.Unknown{
				ContentType: runtime.ContentTypeJSON,
				Raw:         []byte(fmt.Sprintf(`{"guestPath": "%s"}`, tc.guestPath)),
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			// All plugins should be a closer
			if _, ok := p.(io.Closer); !ok {
				t.Fatalf("expected Closer %v", p)
			}
			if _, ok := p.(framework.FilterPlugin); tc.expectFilter != ok {
				t.Fatalf("expected FilterPlugin %v", p)
			}
			if _, ok := p.(framework.ScorePlugin); tc.expectScore != ok {
				t.Fatalf("expected ScorePlugin %v", p)
			}
			if _, ok := p.(framework.BindPlugin); tc.expectBind != ok {
				t.Fatalf("expected BindPlugin %v", p)
			}
		})
	}
}

func TestNewFromConfig(t *testing.T) {
	tests := []struct {
		name          string
		guestPath     string
		expectedError string
	}{
		{
			name:      "valid wasm",
			guestPath: test.PathExampleFilterSimple,
		},
		{
			name:          "not plugin",
			guestPath:     test.PathErrorNotPlugin,
			expectedError: `wasm: guest does not export any plugin functions`,
		},
		{
			name:      "panic on _start",
			guestPath: test.PathErrorPanicOnStart,
			expectedError: `failed to create a guest pool: wasm: instantiate error: panic!
module[panic_on_start-1] function[_start] failed: wasm error: unreachable
wasm stack trace:
	panic_on_start.main()`,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: tc.guestPath})
			if err != nil {
				if want, have := tc.expectedError, err.Error(); want != have {
					t.Fatalf("unexpected error: want %v, have %v", want, have)
				}
			} else if want := tc.expectedError; want != "" {
				t.Fatalf("expected error %v", want)
			}
			if p != nil {
				p.Close()
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name                  string
		guestPath             string
		globals               map[string]int32
		pod                   *v1.Pod
		node                  *v1.Node
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "success: node matches spec.NodeName",
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "unscheduled: bad-node",
			pod:                   test.PodSmall,
			node:                  st.MakeNode().Name("bad-node").Obj(),
			expectedStatusCode:    framework.Unschedulable,
			expectedStatusMessage: "good-node != bad-node",
		},
		{
			name:               "min statusCode",
			guestPath:          test.PathTestFilterFromGlobal,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestPath:          test.PathTestFilterFromGlobal,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestPath:          test.PathErrorPanicOnFilter,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: filter error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_filter.filter() i32`,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			guestPath := tc.guestPath
			if guestPath == "" {
				guestPath = test.PathExampleFilterSimple
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: guestPath})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)
			s := p.Filter(ctx, nil, tc.pod, ni)
			if want, have := tc.expectedStatusCode, s.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMessage, s.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		name                  string
		guestPath             string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeName              string
		expectedScore         int64
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "scored: nodeName equals spec.NodeName",
			pod:                test.PodSmall,
			nodeName:           test.PodSmall.Spec.NodeName,
			expectedScore:      100,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "skipped: bad-node",
			pod:                test.PodSmall,
			nodeName:           "bad-node",
			expectedScore:      0,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "most negative score",
			guestPath:          test.PathTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MinInt32},
			expectedScore:      math.MinInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "min score",
			guestPath:          test.PathTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MinInt32},
			expectedScore:      math.MinInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "max score",
			guestPath:          test.PathTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MaxInt32},
			expectedScore:      math.MaxInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "min statusCode",
			guestPath:          test.PathTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedScore:      0,
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestPath:          test.PathTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedScore:      0,
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestPath:          test.PathErrorPanicOnScore,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: score error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_score.score() i64`,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			guestPath := tc.guestPath
			if guestPath == "" {
				guestPath = test.PathExampleScoreSimple
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: guestPath})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			score, status := p.Score(ctx, nil, tc.pod, tc.nodeName)
			if want, have := tc.expectedScore, score; want != have {
				t.Fatalf("unexpected score: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}
