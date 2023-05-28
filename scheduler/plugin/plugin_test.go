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
	"io"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/testdata"
)

const exampleWasmPath = "../example/main.wasm"

func Test_getOrCreateGuest(t *testing.T) {
	p, err := testdata.NewPluginExampleFilterSimple()
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl, ok := wasm.NewTestWasmPlugin(p)
	if !ok {
		t.Fatalf("failed to cast plugin to wasmPlugin: %v", ok)
	}

	ctx := context.Background()
	uid := types.UID("test-uid")
	differentuid := types.UID("test-uid")

	g, err := pl.GetOrCreateGuest(ctx, uid)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g == nil {
		t.Fatalf("got nil guest instance")
	}

	// this should creat new guest instance because we pass the different podUID.
	g, err = pl.GetOrCreateGuest(ctx, differentuid)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g == nil {
		t.Fatalf("got nil guest instance")
	}

	// remove guestModule to make sure that the next getOrCreateGuest() doesn't try to create new instance.
	pl.ClearGuestModule()

	// this should return the same guest instance as the previous one because we pass the same podUID.
	g, err = pl.GetOrCreateGuest(ctx, uid)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g == nil {
		t.Fatalf("got nil guest instance")
	}
}

// Test_guestPool_assignedToBindingPod tests that the assignedToBindingPod field is set correctly.
func Test_guestPool_assignedToBindingPod(t *testing.T) {
	p, err := testdata.NewPluginExampleFilterSimple()
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl, ok := wasm.NewTestWasmPlugin(p)
	if !ok {
		t.Fatalf("failed to cast plugin to wasmPlugin: %v", ok)
	}

	ctx := context.Background()
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
	p, err := testdata.NewPluginExampleFilterSimple()
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	pl, ok := wasm.NewTestWasmPlugin(p)
	if !ok {
		t.Fatalf("failed to cast plugin to wasmPlugin: %v", ok)
	}

	ctx := context.Background()
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

func TestFilter(t *testing.T) {
	tests := []struct {
		name         string
		pod          *v1.Pod
		node         *v1.Node
		expectedCode framework.Code
	}{
		{
			name:         "success: node is match with spec.NodeName",
			pod:          testdata.PodSmall,
			node:         testdata.NodeSmall,
			expectedCode: framework.Success,
		},
		{
			name:         "filtered: bad-node",
			pod:          testdata.PodSmall,
			node:         st.MakeNode().Name("bad-node").Obj(),
			expectedCode: framework.Unschedulable,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			p, err := testdata.NewPluginExampleFilterSimple()
			if err != nil {
				t.Fatalf("failed to create plugin: %v", err)
			}
			defer p.(io.Closer).Close()

			ni := framework.NewNodeInfo()
			ni.SetNode(tc.node)
			s := p.(framework.FilterPlugin).Filter(context.Background(), nil, tc.pod, ni)
			if s.Code() != tc.expectedCode {
				t.Fatalf("unexpected code: got %v, expected %v, got reason: %v", s.Code(), tc.expectedCode, s.Message())
			}
		})
	}
}
