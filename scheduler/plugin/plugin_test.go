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
	"bytes"
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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
	p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: test.URLTestCycleState}, nil)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl := wasm.NewTestWasmPlugin(p)
	pod := st.MakePod().UID(uuid.New().String()).Name("test-pod").Node("good-node").Obj()
	nextPod := st.MakePod().UID(uuid.New().String()).Name("test-pod2").Node("good-node").Obj()

	_, status := pl.PreFilter(ctx, nil, pod)
	if !status.IsSuccess() {
		t.Fatalf("prefilter failed: %v", status.Reasons())
	}

	ni := framework.NewNodeInfo()
	ni.SetNode(test.NodeSmall)
	status = pl.Filter(ctx, nil, pod, ni)
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status.Reasons())
	}

	if pl.GetScheduledPodUID() != pod.UID {
		t.Fatalf("expected scheduledPodUID to be %v, have %v", pod.UID, pl.GetScheduledPodUID())
	}

	// pod is going to the binding cycle.
	status, _ = pl.Permit(ctx, nil, pod, "node")
	if !status.IsSuccess() {
		t.Fatalf("permit failed: %v", status.Reasons())
	}

	if len(pl.GetBindingCycles()) != 1 {
		t.Fatalf("expected bindingCycles to have 1 entry for `pod`, have %v", len(pl.GetBindingCycles()))
	}

	// another scheduling cycle for nextPod is started.

	_, status = pl.PreFilter(ctx, nil, nextPod)
	if !status.IsSuccess() {
		t.Fatalf("PreFilter failed: %v", status.Reasons())
	}

	if want, have := nextPod.UID, pl.GetScheduledPodUID(); want != have {
		t.Fatalf("unexpected pod UID: want %v, have %v", want, have)
	}

	status = pl.Filter(ctx, nil, pod, ni)
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status.Reasons())
	}

	status, _ = pl.Permit(ctx, nil, nextPod, "node")
	if !status.IsSuccess() {
		t.Fatalf("filter failed: %v", status.Reasons())
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

	// nextPod is going to PreBind process.
	status = pl.PreBind(ctx, nil, nextPod, "node")
	if !status.IsSuccess() {
		t.Fatalf("prebind failed: %v", status.Reasons())
	}

	// nextPod is going to Bind process.
	status = pl.Bind(ctx, nil, nextPod, "node")
	if !status.IsSuccess() {
		t.Fatalf("bind failed: %v", status.Reasons())
	}

	// nextPod is rejected in the binding cycle.
	pl.PostBind(ctx, nil, nextPod, "node")
	if len(pl.GetBindingCycles()) != 0 {
		t.Fatalf("expected bindingCycles to have 0 entry, have %v", len(pl.GetBindingCycles()))
	}
}

// Test_guestPool_assignedToSchedulingPod tests that the scheduledPodUID is assigned during PreFilter expectedly.
func Test_guestPool_assignedToSchedulingPod(t *testing.T) {
	p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: test.URLTestCycleState}, nil)
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
			guestURL:      test.URLExampleAdvanced,
			expectedScore: true,
		},
		{
			name:          "score",
			guestURL:      test.URLErrorPanicOnScore,
			expectedScore: true,
		},
		{
			name:            "all",
			guestURL:        test.URLExampleNodeNumber,
			expectedFilter:  true,
			expectedScore:   true,
			expectedReserve: true,
			expectedBind:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := wasm.PluginFactory("wasm")(context.Background(), &runtime.Unknown{
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
				t.Fatalf("unexpected FilterPlugin %v", p)
			}
			if _, ok := p.(wasm.ScorePlugin); tc.expectedScore != ok {
				t.Fatalf("unexpected ScorePlugin %v", p)
			}
			if _, ok := p.(wasm.ReservePlugin); tc.expectedReserve != ok {
				t.Fatalf("unexpected ReservePlugin %v", p)
			}
			if _, ok := p.(wasm.BindPlugin); tc.expectedBind != ok {
				t.Fatalf("unexpected BindPlugin %v", p)
			}
		})
	}
}

func TestNewFromConfig(t *testing.T) {
	uri, _ := url.ParseRequestURI(test.URLTestFilter)
	bytes, _ := os.ReadFile(uri.Path)
	_, file := path.Split(uri.Path)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+file {
			_, _ = w.Write(bytes)
		}
	}))
	t.Cleanup(ts.Close)

	type testcase struct {
		name          string
		guestURL      string
		expectedError string
	}
	tests := []testcase{
		{
			name:     "file: valid",
			guestURL: test.URLTestFilter,
		},
		{
			name:     "http: valid",
			guestURL: ts.URL + "/" + file,
		},
		{
			name:          "missing guestURL",
			expectedError: "wasm: guestURL is required",
		},
		{
			name:          "invalid guestURL",
			guestURL:      "c:\\foo.wasm",
			expectedError: "wasm: error reading guestURL c:\\foo.wasm: unsupported URL scheme: c",
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: tc.guestURL}, nil)
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
}

func TestEnqueue(t *testing.T) {
	tests := []struct {
		name     string
		guestURL string
		args     []string
		expected []framework.ClusterEventWithHint
	}{
		{
			name:     "success: 0",
			expected: wasm.AllClusterEvents,
		},
		{
			name: "success: 1",
			args: []string{"test", "1"},
			expected: []framework.ClusterEventWithHint{
				{Event: framework.ClusterEvent{Resource: framework.PersistentVolume, ActionType: framework.Delete}},
			},
		},
		{
			name: "success: 2",
			args: []string{"test", "2"},
			expected: []framework.ClusterEventWithHint{
				{Event: framework.ClusterEvent{Resource: framework.Node, ActionType: framework.Add}},
				{Event: framework.ClusterEvent{Resource: framework.PersistentVolume, ActionType: framework.Delete}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestCycleState
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
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

		p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL}, nil)
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
			expectedResult:     &framework.PreFilterResult{NodeNames: sets.New("good-node")},
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

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args, GuestConfig: tc.guestConfig}, nil)
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
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
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

			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, &test.FakeHandle{
				SharedLister: &test.FakeSharedLister{
					NodeInfoLister: &test.FakeNodeInfoLister{
						Nodes: []*framework.NodeInfo{ni},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			ni = framework.NewNodeInfo()
			ni.SetNode(tc.node)
			s := p.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)
			if want, have := tc.expectedStatusCode, s.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d (message: %v)", want, have, s.Message())
			}
			if want, have := tc.expectedStatusMessage, s.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v (message: %v)", want, have, s.Message())
			}
		})
	}
}

func TestPostFilter(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeToStatusMap       map[string]*framework.Status
		expectedResult        *framework.PostFilterResult
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "success",
			args:               []string{"test", "postFilter"},
			pod:                test.PodSmall,
			nodeToStatusMap:    map[string]*framework.Status{test.NodeSmallName: framework.NewStatus(framework.Success, "")},
			expectedResult:     &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "good-node", NominatingMode: framework.ModeOverride}},
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "unschedulable",
			args:                  []string{"test", "postFilter"},
			pod:                   test.PodSmall,
			nodeToStatusMap:       map[string]*framework.Status{test.NodeSmallName: framework.NewStatus(framework.Unschedulable, "")},
			expectedResult:        &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "good-node", NominatingMode: framework.ModeNoop}},
			expectedStatusMessage: "good-node is unschedulable",
			expectedStatusCode:    framework.UnschedulableAndUnresolvable,
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPostFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedResult:     &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "", NominatingMode: 0}},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPostFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedResult:     &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "", NominatingMode: 0}},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "min nominatingMode",
			guestURL:           test.URLTestPostFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"nominating_mode": math.MinInt32},
			expectedResult:     &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "", NominatingMode: math.MinInt32}},
			expectedStatusCode: framework.Success,
		},
		{
			name:               "max nominatingMode",
			guestURL:           test.URLTestPostFilterFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"nominating_mode": math.MaxInt32},
			expectedResult:     &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: "", NominatingMode: math.MaxInt32}},
			expectedStatusCode: framework.Success,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPostFilter,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: postfilter error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_postfilter.$1() i64`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestFilter
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			result, status := p.(framework.PostFilterPlugin).PostFilter(ctx, nil, tc.pod, tc.nodeToStatusMap)
			if want, have := tc.expectedResult, result; !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected result: want %#v, have %#v", want.NominatingInfo, have.NominatingInfo)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
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

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
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

			nodeInfos := make([]*framework.NodeInfo, len(tc.nodes))
			for i, n := range tc.nodes {
				ni := framework.NewNodeInfo()
				ni.SetNode(n)
				nodeInfos[i] = ni
			}
			status := p.(framework.PreScorePlugin).PreScore(ctx, nil, tc.pod, nodeInfos)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
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

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
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
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeScoreList         framework.NodeScoreList
		expectedStatusCode    framework.Code
		expectedStatusMessage string
		expectedError         string
		expectedNodeScoreList framework.NodeScoreList
	}{
		{
			name: "normalizescore: multiply nodeScore by 100",
			args: []string{"test", "scoreExtensions"},
			pod:  test.PodSmall,
			nodeScoreList: framework.NodeScoreList{
				{Name: test.NodeSmall.Name, Score: 100},
			},
			expectedStatusCode: framework.Success,
			expectedNodeScoreList: framework.NodeScoreList{
				{Name: test.NodeSmall.Name, Score: 10000},
			},
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestScoreExtensionsFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestScoreExtensionsFromGlobal,
			pod:                test.PodSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnScoreExtensions,
			pod:                test.PodSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: normalizescore error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_scoreextensions.$1() i32`,
		},
		{
			name:               "missing score",
			guestURL:           test.URLErrorScoreExtensionsWithoutScore,
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

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
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

			status := p.(framework.ScoreExtensions).NormalizeScore(ctx, nil, tc.pod, tc.nodeScoreList)
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want \n%v, have \n%v", want, have)
			}
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if tc.expectedNodeScoreList != nil {
				if tc.expectedNodeScoreList[0] != tc.nodeScoreList[0] {
					// This test is just for "normalizescore: multiply nodeScore by 100" case.
					t.Fatalf("unexpected nodeScoreList: want %v, have %v", tc.expectedNodeScoreList[0], tc.nodeScoreList[0])
				}
			}
		})
	}
}

func TestReserve(t *testing.T) {
	tests := []struct {
		name                   string
		guestURL               string
		args                   []string
		globals                map[string]int32
		pod                    *v1.Pod
		nodeName               string
		expectedStatusCode     framework.Code
		expectedStatusMessage  string
		expectedUnreserveError string
	}{
		{
			name:               "Success",
			pod:                test.PodSmall,
			nodeName:           "good",
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			pod:                   test.PodSmall,
			nodeName:              "bad",
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "name is bad",
		},
		{
			name:     "reachable: flag is 0",
			guestURL: test.URLTestReserveFromGlobal,
			pod:      test.PodSmall,
			nodeName: test.NodeSmall.Name,
			globals:  map[string]int32{"flag": 0},
		},
		{
			name:     "unreachable: flag is 1",
			guestURL: test.URLTestReserveFromGlobal,
			pod:      test.PodSmall,
			nodeName: test.NodeSmall.Name,
			globals:  map[string]int32{"flag": 1},
			expectedUnreserveError: `"failed unreserve" err=<
	wasm: unreserve error: wasm error: unreachable
	wasm stack trace:
		reserve_from_global.$1()
 >`,
		},
		{
			name:                  "panic",
			guestURL:              test.URLErrorPanicOnReserve,
			pod:                   test.PodSmall,
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "wasm: reserve error: panic!\nwasm error: unreachable\nwasm stack trace:\n\tpanic_on_reserve.$1() i32",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestReserve
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			cycleState := framework.NewCycleState()
			status := p.(framework.ReservePlugin).Reserve(ctx, cycleState, tc.pod, tc.nodeName)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
			if !status.IsSuccess() {
				// If Reserve failed, Unreserve is not valuable to test.
				return
			}

			// Because Unreserve doesn't return any values, we use klog's error for testing.
			klogErr, err := captureStderr(func() {
				p.(framework.ReservePlugin).Unreserve(ctx, cycleState, tc.pod, tc.nodeName)
			})
			if err != nil {
				t.Fatal(err)
			}
			// if want, have := tc.expectedUnreserveError, extractMessage(klogErr); cmp.Diff(x, y, opts) != have {
			if diff := cmp.Diff(tc.expectedUnreserveError, extractMessage(klogErr)); diff != "" {
				t.Fatalf("unexpected unreserve error: %s", diff)
			}
		})
	}
}

func TestPermit(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeName              string
		expectedStatusCode    framework.Code
		expectedStatusMessage string
		expectedTimeout       time.Duration
	}{
		{
			name:               "Success",
			pod:                test.PodSmall,
			nodeName:           "good",
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			pod:                   test.PodSmall,
			nodeName:              "bad",
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "name is bad",
		},
		{
			name:                  "Wait",
			pod:                   test.PodSmall,
			nodeName:              "wait",
			expectedStatusCode:    framework.Wait,
			expectedStatusMessage: "name is wait",
			expectedTimeout:       10 * time.Second,
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPermitFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPermitFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:                  "panic",
			guestURL:              test.URLErrorPanicOnPermit,
			pod:                   test.PodSmall,
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "wasm: permit error: panic!\nwasm error: unreachable\nwasm stack trace:\n\tpanic_on_permit.$1() i64",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestPermit
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			cycleState := framework.NewCycleState()
			status, timeout := p.(framework.PermitPlugin).Permit(ctx, cycleState, tc.pod, tc.nodeName)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
			if want, have := tc.expectedTimeout, timeout; want != have {
				t.Fatalf("unexpected timeout: want %v, have %v", want, have)
			}
		})
	}
}

func TestPreBind(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeName              string
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "Success",
			args:               []string{"test", "preBind"},
			pod:                test.PodSmall,
			nodeName:           "good",
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			args:                  []string{"test", "preBind"},
			pod:                   test.PodSmall,
			nodeName:              "bad",
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "name is bad",
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPreBindFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPreBindFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPreBind,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: prebind error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_prebind.$1() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestBind
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			pl := wasm.NewTestWasmPlugin(p)
			if len(tc.globals) > 0 {
				pl.SetGlobals(tc.globals)
			}
			pl.CreateGuestInBindingGuestPool(tc.pod.UID)

			status := p.(framework.PreBindPlugin).PreBind(ctx, nil, tc.pod, tc.nodeName)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestBind(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		nodeName              string
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "Success",
			args:               []string{"test", "bind"},
			pod:                test.PodSmall,
			nodeName:           "good",
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			args:                  []string{"test", "bind"},
			pod:                   test.PodSmall,
			nodeName:              "bad",
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "name is bad",
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestBindFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestBindFromGlobal,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnBind,
			pod:                test.PodSmall,
			nodeName:           test.NodeSmall.Name,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: bind error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_bind.$1() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestBind
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			pl := wasm.NewTestWasmPlugin(p)
			if len(tc.globals) > 0 {
				pl.SetGlobals(tc.globals)
			}
			pl.CreateGuestInBindingGuestPool(tc.pod.UID)

			status := p.(framework.BindPlugin).Bind(ctx, nil, tc.pod, tc.nodeName)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestPostBind(t *testing.T) {
	tests := []struct {
		name          string
		guestURL      string
		args          []string
		globals       map[string]int32
		pod           *v1.Pod
		nodeName      string
		expectedError string
	}{
		{
			name:     "Success",
			args:     []string{"test", "postBind"},
			pod:      test.PodSmall,
			nodeName: "good",
		},
		{
			name:     "Error",
			args:     []string{"test", "postBind"},
			pod:      test.PodSmall,
			nodeName: "bad",
			expectedError: `"failed postbind" err=<
	wasm: postbind error: panic: name is bad
	
	wasm error: unreachable
	wasm stack trace:
		main.runtime._panic(i32,i32)
		main.postbind()
 >`,
		},
		{
			name:     "reachable: flag is 0",
			guestURL: test.URLTestPostBindFromGlobal,
			pod:      test.PodSmall,
			nodeName: test.NodeSmall.Name,
			globals:  map[string]int32{"flag": 0},
		},
		{
			name:     "unreachable: flag is 1",
			guestURL: test.URLTestPostBindFromGlobal,
			pod:      test.PodSmall,
			nodeName: test.NodeSmall.Name,
			globals:  map[string]int32{"flag": 1},
			expectedError: `"failed postbind" err=<
	wasm: postbind error: wasm error: unreachable
	wasm stack trace:
		postbind_from_global.$0()
 >`,
		},
		{
			name:     "panic",
			guestURL: test.URLErrorPanicOnPostBind,
			pod:      test.PodSmall,
			nodeName: test.NodeSmall.Name,
			expectedError: `"failed postbind" err=<
	wasm: postbind error: panic!
	wasm error: unreachable
	wasm stack trace:
		panic_on_postbind.$1()
 >`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestBind
			}

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			pl := wasm.NewTestWasmPlugin(p)
			if len(tc.globals) > 0 {
				pl.SetGlobals(tc.globals)
			}
			pl.CreateGuestInBindingGuestPool(tc.pod.UID)

			// Because postBind doesn't return any values, we use klog's error for testing.
			klogErr, err := captureStderr(func() {
				p.(framework.PostBindPlugin).PostBind(ctx, nil, tc.pod, tc.nodeName)
			})
			if err != nil {
				t.Fatalf("got an error during captureStderr %v", err)
			}
			if want, have := tc.expectedError, extractMessage(klogErr); want != have {
				t.Fatalf("unexpected log: want%v, have%v", want, have)
			}
		})
	}
}

func TestAddPod(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		podInfo               framework.PodInfo
		node                  *v1.Node
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "Success",
			args:               []string{"test", "preFilterExtensions"},
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{Pod: st.MakePod().Name("good-pod").Obj()},
			node:               test.NodeSmall,
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			args:                  []string{"test", "preFilterExtensions"},
			pod:                   test.PodSmall,
			podInfo:               framework.PodInfo{Pod: st.MakePod().Name("bad-pod").Obj()},
			node:                  st.MakeNode().Name("bad").Obj(),
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "Node name is bad and PodInfo name is bad-pod",
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPreFilterExtensionsFromGlobal,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPreFilterExtensionsFromGlobal,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPreFilterExtensions,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: addpod error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_prefilterextensions.$1() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestFilter
			}

			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, &test.FakeHandle{
				SharedLister: &test.FakeSharedLister{
					NodeInfoLister: &test.FakeNodeInfoLister{
						Nodes: []*framework.NodeInfo{ni},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			status := p.(framework.PreFilterExtensions).AddPod(ctx, nil, tc.pod, &tc.podInfo, ni)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestRemovePod(t *testing.T) {
	tests := []struct {
		name                  string
		guestURL              string
		args                  []string
		globals               map[string]int32
		pod                   *v1.Pod
		podInfo               framework.PodInfo
		node                  *v1.Node
		expectedStatusCode    framework.Code
		expectedStatusMessage string
	}{
		{
			name:               "Success",
			args:               []string{"test", "preFilterExtensions"},
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{Pod: st.MakePod().Name("good-pod").Obj()},
			node:               test.NodeSmall,
			expectedStatusCode: framework.Success,
		},
		{
			name:                  "Error",
			args:                  []string{"test", "preFilterExtensions"},
			pod:                   test.PodSmall,
			podInfo:               framework.PodInfo{Pod: st.MakePod().Name("bad-pod").Obj()},
			node:                  st.MakeNode().Name("bad").Obj(),
			expectedStatusCode:    framework.Error,
			expectedStatusMessage: "Node name is bad and PodInfo name is bad-pod",
		},
		{
			name:               "min statusCode",
			guestURL:           test.URLTestPreFilterExtensionsFromGlobal,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MinInt32},
			expectedStatusCode: math.MinInt32,
		},
		{
			name:               "max statusCode",
			guestURL:           test.URLTestPreFilterExtensionsFromGlobal,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			globals:            map[string]int32{"status_code": math.MaxInt32},
			expectedStatusCode: math.MaxInt32,
		},
		{
			name:               "panic",
			guestURL:           test.URLErrorPanicOnPreFilterExtensions,
			pod:                test.PodSmall,
			podInfo:            framework.PodInfo{},
			node:               test.NodeSmall,
			expectedStatusCode: framework.Error,
			expectedStatusMessage: `wasm: removepod error: panic!
wasm error: unreachable
wasm stack trace:
	panic_on_prefilterextensions.$2() i32`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestFilter
			}
			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)

			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, &test.FakeHandle{
				SharedLister: &test.FakeSharedLister{
					NodeInfoLister: &test.FakeNodeInfoLister{
						Nodes: []*framework.NodeInfo{ni},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			if len(tc.globals) > 0 {
				pl := wasm.NewTestWasmPlugin(p)
				pl.SetGlobals(tc.globals)
			}

			status := p.(framework.PreFilterExtensions).RemovePod(ctx, nil, tc.pod, &tc.podInfo, ni)
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %d, have %d", want, have)
			}
			if want, have := tc.expectedStatusMessage, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

// This test checks whether framework.handle.EventRecorder.Eventf can be called within wasm file.
func TestEventf(t *testing.T) {
	tests := []struct {
		name        string
		guestURL    string
		pod         *v1.Pod
		expectedMsg string
	}{
		{
			name:        "Test for skipping preScore using URLExampleNodeNumber with handle set via plugin.Set)",
			guestURL:    test.URLExampleNodeNumber,
			pod:         test.PodSmall,
			expectedMsg: "good-pod PreScore not match lastNumber Skip ",
		},
		{
			name:        "Test for skipping preScore using URLExampleAdvanced with handle set via prescore.SetPlugin)",
			guestURL:    test.URLExampleAdvanced,
			pod:         test.PodSmall,
			expectedMsg: "good-pod PreScore not match lastNumber Skip ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			if guestURL == "" {
				guestURL = test.URLTestScore
			}
			recorder := &test.FakeRecorder{EventMsg: ""}
			handle := &test.FakeHandle{Recorder: recorder}
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL}, handle)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			// Not use status for this test.
			_ = p.(framework.PreScorePlugin).PreScore(ctx, nil, tc.pod, nil)

			if want, have := tc.expectedMsg, recorder.EventMsg; want != have {
				t.Fatalf("unexpected Event Msg: %v != %v", want, have)
			}
		})
	}
}

// This test checks whether framework.handle.RejectWaitingPod can be called within wasm file.
func TestRejectWaitingPod(t *testing.T) {
	tests := []struct {
		name               string
		guestURL           string
		pod                *v1.Pod
		args               []string
		expectedUID        types.UID
		expectedStatusCode framework.Code
		expectedStatusMsg  string
	}{
		{
			name:               "Pod is not rejected",
			guestURL:           test.URLTestHandle,
			pod:                test.PodSmall,
			args:               []string{"test", "rejectWaitingPod"},
			expectedUID:        test.PodSmall.GetUID(),
			expectedStatusCode: framework.Success,
			expectedStatusMsg:  "",
		},
		{
			name:               "Pod is rejected",
			guestURL:           test.URLTestHandle,
			pod:                test.PodForHandleTest,
			args:               []string{"test", "rejectWaitingPod"},
			expectedUID:        test.PodForHandleTest.GetUID(),
			expectedStatusCode: framework.Skip,
			expectedStatusMsg:  "UID is handle-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			recorder := &test.FakeRecorder{EventMsg: ""}
			handle := &test.FakeHandle{Recorder: recorder}
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, handle)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()
			ni := framework.NewNodeInfo()
			ni.SetNode(test.NodeSmall)
			status := p.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)
			if want, have := tc.expectedUID, handle.RejectWaitingPodValue; want != have {
				t.Fatalf("unexpected uid: %v != %v", want, have)
			}
			if want, have := tc.expectedStatusCode, status.Code(); want != have {
				t.Fatalf("unexpected status code: want %v, have %v", want, have)
			}
			if want, have := tc.expectedStatusMsg, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

func TestGetWaitingPod(t *testing.T) {
	tests := []struct {
		name               string
		guestURL           string
		pod                *v1.Pod
		args               []string
		expectedUID        types.UID
		expectedWaitingPod framework.WaitingPod
		expectedStatusCode framework.Code
		expectedStatusMsg  string
	}{
		{
			name:               "Pod is not returned",
			guestURL:           test.URLTestHandle,
			pod:                test.PodForHandleTest,
			args:               []string{"test", "getWaitingPod"},
			expectedUID:        "non-existent-uid",
			expectedWaitingPod: nil,
			expectedStatusCode: framework.Error,
			expectedStatusMsg:  "No waiting pod found for UID: handle-test",
		},
		{
			name:               "Pod is returned",
			guestURL:           test.URLTestHandle,
			pod:                test.PodSmall,
			args:               []string{"test", "getWaitingPod"},
			expectedUID:        "handle-test",
			expectedWaitingPod: makeTestWaitingPod(test.PodForHandlePod, map[string]*time.Timer{}),
			expectedStatusCode: framework.Success,
			expectedStatusMsg:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guestURL := tc.guestURL
			recorder := &test.FakeRecorder{EventMsg: ""}
			handle := &test.FakeHandle{Recorder: recorder}

			// Create a new Wasm plugin instance.
			p, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: guestURL, Args: tc.args}, handle)
			if err != nil {
				t.Fatal(err)
			}
			defer p.(io.Closer).Close()

			// Create node info and set up a node.
			ni := framework.NewNodeInfo()
			ni.SetNode(test.NodeSmall)

			status := p.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)

			got := handle.GetWaitingPodValue
			want := tc.expectedWaitingPod

			if want == nil {
				if got != nil {
					t.Fatalf("expected no pod, but got: %v", got)
				}
			} else {
				// Compare the pod's UID and pendingPlugins map
				if !comparePods(got.GetPod(), want.GetPod()) {
					t.Fatalf("unexpected pod: got %+v, want %+v", got.GetPod(), want.GetPod())
				}

				if !reflect.DeepEqual(got.GetPendingPlugins(), want.GetPendingPlugins()) {
					t.Fatalf("unexpected pending plugins: got %+v, want %+v", got.GetPendingPlugins(), want.GetPendingPlugins())
				}
			}

			//if want, have := tc.expectedStatusCode, status.Code(); want != have {
			//	t.Fatalf("unexpected status code: want %v, have %v", want, have)
			//}

			if want, have := tc.expectedStatusMsg, status.Message(); want != have {
				t.Fatalf("unexpected status message: want %v, have %v", want, have)
			}
		})
	}
}

// Extracts and trims the actual log message from a formatted klog string
// (klog includes timestamp before actual log message)
func extractMessage(log string) string {
	parts := strings.SplitN(log, "]", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// captureStderr temporarily redirects the standard error output to capture any data written to it.
// This function is particularly useful for capturing klog's error output during tests.
// It takes a function f, executes it, and captures anything written to stderr during its execution.
// After the function execution, it restores the original stderr and returns the captured output as a string.
func captureStderr(f func()) (string, error) {
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = originalStderr

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	r.Close()

	return buf.String(), nil
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

type waitingPod struct {
	pod            *v1.Pod
	pendingPlugins map[string]*time.Timer
	mu             sync.RWMutex
}

func (wp *waitingPod) GetPod() *v1.Pod {
	return wp.pod
}

func (wp *waitingPod) GetPendingPlugins() []string {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	var plugins []string
	for plugin := range wp.pendingPlugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (wp *waitingPod) Allow(pluginName string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if timer, ok := wp.pendingPlugins[pluginName]; ok {
		timer.Stop()
		delete(wp.pendingPlugins, pluginName)
	}
}

func (wp *waitingPod) Reject(pluginName, msg string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if timer, ok := wp.pendingPlugins[pluginName]; ok {
		timer.Stop()
		delete(wp.pendingPlugins, pluginName)
	}
}

func makeTestWaitingPod(pod *v1.Pod, plugins map[string]*time.Timer) framework.WaitingPod {
	return &waitingPod{
		pod:            pod,
		pendingPlugins: plugins,
	}
}

// comparePods compares the UIDs of two v1.Pod objects
func comparePods(pod1, pod2 *v1.Pod) bool {
	if pod1 == nil || pod2 == nil {
		return pod1 == pod2
	}
	return pod1.UID == pod2.UID && pod1.Name == pod2.Name && pod1.Namespace == pod2.Namespace
}
