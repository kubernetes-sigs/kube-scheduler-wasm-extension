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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/google/uuid"
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

// Test_guestPool_bindingCycles tests that the bindingCycles field is set correctly.
func Test_guestPool_bindingCycles(t *testing.T) {
	p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: test.URLTestAll})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl := wasm.NewTestWasmPlugin(p)
	pod := st.MakePod().UID(uuid.New().String()).Name("test-pod").Node("good-node").Obj()
	nextPod := st.MakePod().UID(uuid.New().String()).Name("test-pod2").Node("good-node").Obj()

	_, status := pl.PreFilter(ctx, nil, pod)
	if !status.IsSuccess() {
		t.Fatalf("prefilter failed: %v", status)
	}

	if pl.GetScheduledPodUID() != pod.UID {
		t.Fatalf("expected scheduledPodUID to be %v, have %v", pod.UID, pl.GetScheduledPodUID())
	}

	// pod is going to the binding cycle.
	status, _ = pl.Permit(ctx, nil, pod, "node")
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status)
	}

	if len(pl.GetBindingCycles()) != 1 {
		t.Fatalf("expected bindingCycles to have 1 entry for `pod`, have %v", len(pl.GetBindingCycles()))
	}

	// another scheduling cycle for nextPod is started.

	_, status = pl.PreFilter(ctx, nil, nextPod)
	if !status.IsSuccess() {
		t.Fatalf("PreFilter failed: %v", status)
	}

	if want, have := nextPod.UID, pl.GetScheduledPodUID(); want != have {
		t.Fatalf("unexpected pod UID: want %v, have %v", want, have)
	}

	status, _ = pl.Permit(ctx, nil, nextPod, "node")
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status)
	}

	if len(pl.GetBindingCycles()) != 2 {
		t.Fatalf("expected bindingCycles to have 2 entry for `pod`, have %v", len(pl.GetBindingCycles()))
	}

	// make sure that the bindingCycles has entries for both `pod` and `nextPod`.

	registeredPodUIDs := sets.New[types.UID]()
	for podUID := range pl.GetBindingCycles() {
		registeredPodUIDs.Insert(podUID)
	}
	if !registeredPodUIDs.Has(pod.UID) || !registeredPodUIDs.Has(nextPod.UID) {
		t.Fatalf("expected bindingCycles to have entries for `pod` and `nextPod`, but have %v", registeredPodUIDs)
	}

	// pod is rejected in the binding cycle.
	pl.Unreserve(ctx, nil, pod, "node")
	bindingCycles := pl.GetBindingCycles()
	if len(bindingCycles) != 1 {
		t.Fatalf("expected bindingCycles to have 1 entry for `nextPod`, have %v", bindingCycles)
	}
	if _, ok := bindingCycles[nextPod.UID]; !ok {
		t.Fatalf("expected bindingCycles to have entry for `nextPod`, have %v", bindingCycles)
	}

	// nextPod is rejected in the binding cycle.
	pl.PostBind(ctx, nil, nextPod, "node")
	if len(pl.GetBindingCycles()) != 0 {
		t.Fatalf("expected bindingCycles to have 0 entry, have %v", len(pl.GetBindingCycles()))
	}
}

// Test_guestPool_assignedToSchedulingPod tests that the scheduledPodUID is assigned during PreFilter expectedly.
func Test_guestPool_assignedToSchedulingPod(t *testing.T) {
	p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: test.URLTestAll})
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl := wasm.NewTestWasmPlugin(p)
	pod := st.MakePod().UID(uuid.New().String()).Name("test-pod").Node("good-node").Obj()
	nextPod := st.MakePod().UID(uuid.New().String()).Name("test-pod2").Node("good-node").Obj()

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

	if pl.GetScheduledPodUID() != pod.UID {
		t.Fatalf("expected scheduledPodUID to be %v, have %v", pod.UID, pl.GetScheduledPodUID())
	}

	// PreFilter is called with a different pod, meaning the past scheduling cycle of `pod` is finished.
	pl.PreFilter(ctx, nil, nextPod)

	if want, have := nextPod.UID, pl.GetScheduledPodUID(); want != have {
		t.Fatalf("unexpected pod UID: want %v, have %v", want, have)
	}

	if len(pl.GetFreePool()) != 0 {
		t.Fatal("expected guest instance that is used for `pod` to be reused, but it wasn't")
	}
}

// TestNew_maskInterfaces ensures the type returned by New can be asserted
// against, based on exports in the guest.
func TestNew_maskInterfaces(t *testing.T) {
	tests := []struct {
		name            string
		guestURL        string
		expectedFilter  bool
		expectedScore   bool
		expectedReserve bool
		expectedPermit  bool
		expectedBind    bool
		expectedError   string
	}{
		{
			name:          "not plugin",
			guestURL:      test.URLErrorNotPlugin,
			expectedError: "wasm: guest does not export any plugin functions", // not supported to be only enqueue
		},
		{
			name:           "filter",
			guestURL:       test.URLErrorPanicOnFilter,
			expectedFilter: true,
		},
		{
			name:          "prescore|score",
			guestURL:      test.URLExampleNodeNumber,
			expectedScore: true,
		},
		{
			name:          "score",
			guestURL:      test.URLErrorPanicOnScore,
			expectedScore: true,
		},
		{
			name:           "prefilter|filter|prescore|score",
			guestURL:       test.URLTestAllNoopWat,
			expectedFilter: true,
			expectedScore:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := wasm.New(&runtime.Unknown{
				ContentType: runtime.ContentTypeJSON,
				Raw:         []byte(fmt.Sprintf(`{"guestURL": "%s"}`, tc.guestURL)),
			}, nil)
			if tc.expectedError != "" {
				requireError(t, err, tc.expectedError)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if _, ok := p.(wasm.BasePlugin); !ok {
				t.Fatalf("expecteded BasePlugin %v", p)
			}
			if _, ok := p.(wasm.FilterPlugin); tc.expectedFilter != ok {
				t.Fatalf("didn't expected FilterPlugin %v", p)
			}
			if _, ok := p.(wasm.ScorePlugin); tc.expectedScore != ok {
				t.Fatalf("didn't expected ScorePlugin %v", p)
			}
			if _, ok := p.(wasm.ReservePlugin); tc.expectedReserve != ok {
				t.Fatalf("didn't expected ReservePlugin %v", p)
			}
			if _, ok := p.(wasm.PermitPlugin); tc.expectedPermit != ok {
				t.Fatalf("didn't expected PermitPlugin %v", p)
			}
			if _, ok := p.(wasm.BindPlugin); tc.expectedBind != ok {
				t.Fatalf("didn't expected BindPlugin %v", p)
			}
		})
	}
}

func TestNewFromConfig(t *testing.T) {
	type testcase struct {
		name          string
		guestURL      string
		expectedError string
	}
	tests := []testcase{
		{
			name:     "valid wasm",
			guestURL: test.URLTestFilter,
		},
		{
			name:          "not plugin",
			guestURL:      test.URLErrorNotPlugin,
			expectedError: `wasm: guest does not export any plugin functions`,
		},
		{
			name:     "panic on _start",
			guestURL: test.URLErrorPanicOnStart,
			expectedError: `failed to create a guest pool: wasm: instantiate error: panic!
module[1] function[_start] failed: wasm error: unreachable
wasm stack trace:
	panic_on_start.$2()`,
		},
	}

	testWithURL := func(t *testing.T, tc testcase) {
		t.Run(tc.name, func(t *testing.T) {
			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: tc.guestURL})
			if err != nil {
				if want, have := tc.expectedError, err.Error(); want != have {
					t.Fatalf("unexpected error: want %v, have %v", want, have)
				}
			} else if want := tc.expectedError; want != "" {
				t.Fatalf("expected error %v", want)
			}
			if p != nil {
				p.(io.Closer).Close()
			}
		})
	}

	t.Run("local", func(t *testing.T) {
		for _, tc := range tests {
			testWithURL(t, tc)
		}
	})

	t.Run("remote (http)", func(t *testing.T) {
		for _, tc := range tests {
			uri, _ := url.ParseRequestURI(tc.guestURL)
			bytes, _ := os.ReadFile(uri.Path)
			_, file := path.Split(uri.Path)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/"+file {
					_, _ = w.Write(bytes)
				}
			}))
			defer ts.Close()
			tc.guestURL = ts.URL + "/" + file
			testWithURL(t, tc)
		}
	})
}

func TestEnqueue(t *testing.T) {
	tests := []struct {
		name     string
		guestURL string
		args     []string
		expected []framework.ClusterEvent
	}{
		{
			name:     "success: 0",
			expected: wasm.AllClusterEvents,
		},
		{
			name: "success: 1",
			args: []string{"test", "1"},
			expected: []framework.ClusterEvent{
				{Resource: framework.PersistentVolume, ActionType: framework.Delete},
			},
		},
		{
			name: "success: 2",
			args: []string{"test", "2"},
			expected: []framework.ClusterEvent{
				{Resource: framework.Node, ActionType: framework.Add},
				{Resource: framework.PersistentVolume, ActionType: framework.Delete},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestCycleState
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL, Args: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			clusterEvents := p.(framework.EnqueueExtensions).EventsToRegister()
			if want, have := tc.expected, clusterEvents; !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected node names: want %v, have %v", want, have)
			}
		})
	}

	t.Run("panic", func(t *testing.T) {
		guestURL := test.URLErrorPanicOnEnqueue

		p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL})
		if err != nil {
			t.Fatal(err)
		}
		defer p.(io.Closer).Close()

		if captured := test.CapturePanic(func() {
			_ = p.(framework.EnqueueExtensions).EventsToRegister()
		}); captured == "" {
			t.Fatal("expected to panic")
		}
	})
}

func TestPreFilter(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		guestConfig           string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		expectedResult        *framework.PreFilterResult
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "success: pod has spec.NodeName",
			pod:                test.PodSmall,
			args:               []string{"test", "preFilter"},
			expectedResult:     &framework.PreFilterResult{NodeNames: sets.NewString("good-node")},
			expectedStatusCode: framework.Success,
		},
		{
			name:               "success: pod has no spec.NodeName",
			args:               []string{"test", "preFilter"},
			pod:                &v1.Pod{ObjectMeta: test.PodSmall.ObjectMeta},
			expectedStatusCode: framework.Success,
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPreFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPreFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPreFilter,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: prefilter error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_prefilter.$1() i32`,
		},
		{
			name:               "panic no guestConfig",
			guestURL:           test.URLErrorPanicOnGetConfig,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: prefilter error: wasm error: unreachable
wasm stack trace:
	panic_on_get_config.$2() i32`,
		},
		{ // This only tests that configuration gets assigned
			name:               "panic guestConfig",
			guestURL:           test.URLErrorPanicOnGetConfig,
			guestConfig:        "hello",
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: prefilter error: hello
wasm error: unreachable
wasm stack trace:
	panic_on_get_config.$2() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestFilter
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL, Args: tc.args, GuestConfig: tc.guestConfig})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			nodeNames, status := p.(framework.PreFilterPlugin).PreFilter(ctx, nil, tc.pod)
			if want, have := tc.expectedResult, nodeNames; !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected node names: want %v, have %v", want, have)
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

func TestFilter(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		node                  *v1.Node
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "success: node matches spec.NodeName",
			args:               []string{"test", "filter"},
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "unscheduled: bad-node",
			args:                  []string{"test", "filter"},
			pod:                   test.PodSmall,
			node:                  st.MakeNode().Name("bad-node").Obj(),
			expectedStatusCode:    framework.Unschedulable,
			expectedStatusMessage: "good-node != bad-node",
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestFilterFromGlobal,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestFilterFromGlobal,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnFilter,
			pod:                test.PodSmall,
			node:               test.NodeSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: filter error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_filter.$1() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestFilter
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL, Args: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)
			s := p.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)
			if want, have := tc.expectedStatusCode, s.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMessage, s.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestPreScore(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodes                 []*v1.Node
		expectedStatusCode    framework.Code
		expectedStatusMessage string
		expectedError         string
	}{
		{
			name:               "success: no nodes",
			pod:                test.PodSmall,
			args:               []string{"test", "preScore"},
			expectedStatusCode: 0, // count of nodes
		},
		{
			name:               "success: one node",
			args:               []string{"test", "preScore"},
			nodes:              []*v1.Node{test.NodeSmall},
			pod:                &v1.Pod{ObjectMeta: test.PodSmall.ObjectMeta},
			expectedStatusCode: 1, // count of nodes
		},
		{
			name:               "success: two nodes",
			args:               []string{"test", "preScore"},
			nodes:              []*v1.Node{test.NodeSmall, test.NodeReal},
			pod:                &v1.Pod{ObjectMeta: test.PodSmall.ObjectMeta},
			expectedStatusCode: 2, // count of nodes
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPreScoreFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPreScoreFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPreScore,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: prescore error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_prescore.$1() i32`,
		},
		{
			name:               "missing score",
			guestURL:           test.URLErrorPreScoreWithoutScore,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedError:      `wasm: filter, score, reserve, permit or bind must be exported`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestScore
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL, Args: tc.args})
			if tc.expectedError != "" {
				requireError(t, err, tc.expectedError)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			status := p.(framework.PreScorePlugin).PreScore(ctx, nil, tc.pod, tc.nodes)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeName              string
		expectedScore         int64
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "scored: nodeName equals spec.NodeName",
			args:               []string{"test", "score"},
			pod:                test.PodSmall,
			nodeName:           test.PodSmall.Spec.NodeName,
			expectedScore:      100,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "skipped: bad-node",
			args:               []string{"test", "score"},
			pod:                test.PodSmall,
			nodeName:           "bad-node",
			expectedScore:      0,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "most negative score",
			guestURL:           test.URLTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MinInt32},
			expectedScore:      math.MinInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "min score",
			guestURL:           test.URLTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MinInt32},
			expectedScore:      math.MinInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "max score",
			guestURL:           test.URLTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"score": math.MaxInt32},
			expectedScore:      math.MaxInt32,
			expectedStatusCode: framework.Success,
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedScore:      0,
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestScoreFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedScore:      0,
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnScore,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: score error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_score.$1() i64`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestScore
			}

			p, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestURL: guestURL, Args: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			score, status := p.(framework.ScorePlugin).Score(ctx, nil, tc.pod, tc.nodeName)
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

func requireError(t *testing.T, err error, expectedError string) {
	var have string
	if err != nil {
		have = err.Error()
	}
	if want := expectedError; want != have {
		t.Fatalf("unexpected error: want %v, have %v", want, have)
	}
}
