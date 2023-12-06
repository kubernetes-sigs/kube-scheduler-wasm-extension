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

package main

import (
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

func Test_NodeNumber(t *testing.T) {
	tests := []struct {
		name          string
		pod           proto.Pod
		nodeName      string
		expectedMatch bool
	}{
		{name: "nil,empty", pod: &testPod{}, expectedMatch: true},
		{name: "empty,empty", pod: &testPod{}, nodeName: "", expectedMatch: true},
		{name: "empty,letter", pod: &testPod{}, nodeName: "a", expectedMatch: true},
		{name: "empty,digit", pod: &testPod{}, nodeName: "1", expectedMatch: true},
		{name: "letter,letter", pod: &testPod{nodeName: "a"}, nodeName: "a", expectedMatch: true},
		{name: "letter,digit", pod: &testPod{nodeName: "a"}, nodeName: "1", expectedMatch: true},
		{name: "digit,letter", pod: &testPod{nodeName: "1"}, nodeName: "a", expectedMatch: false},
		{name: "digit,digit", pod: &testPod{nodeName: "1"}, nodeName: "1", expectedMatch: true},
		{name: "digit,different digit", pod: &testPod{nodeName: "1"}, nodeName: "2", expectedMatch: false},
	}

	for _, reverse := range []bool{false, true} {
		for _, tc := range tests {
			name := tc.name
			expectedMatch := tc.expectedMatch
			if reverse {
				name += ",reverse"
				expectedMatch = !expectedMatch
			}
			t.Run(name, func(t *testing.T) {
				plugin := &NodeNumber{reverse: reverse}
				state := testCycleState{}

				status := plugin.PreScore(state, tc.pod, nil)
				if status != nil {
					t.Fatalf("unexpected status: %v", status)
				}

				score, status := plugin.Score(state, nil, tc.nodeName)
				if status != nil {
					t.Fatalf("unexpected status: %v", status)
				}

				if expectedMatch {
					if want, have := int32(10), score; want != have {
						t.Fatalf("unexpected score: %v != %v", want, have)
					}
				} else {
					if want, have := int32(0), score; want != have {
						t.Fatalf("unexpected score: %v != %v", want, have)
					}
				}
			})
		}
	}
}

func Test_lastNumber(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedDigit uint8
		expectedOk    bool
	}{
		{name: "empty", input: ""},
		{name: "not digit", input: "a"},
		{name: "unicode", input: "ó"},
		{name: "middle digit", input: "a1a"},
		{name: "digit after letter", input: "a1", expectedDigit: 1, expectedOk: true},
		{name: "digit after digit", input: "12", expectedDigit: 2, expectedOk: true},
		{name: "digit after unicode", input: "ó2", expectedDigit: 2, expectedOk: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := lastNumber(tc.input)
			if want, have := tc.expectedDigit, d; want != have {
				t.Fatalf("unexpected digit: %v != %v", want, have)
			}
			if want, have := tc.expectedOk, ok; want != have {
				t.Fatalf("unexpected ok: %v != %v", want, have)
			}
		})
	}
}

var _ api.CycleState = testCycleState{}

type testCycleState map[string]any

func (c testCycleState) Read(key string) (val any, ok bool) {
	val, ok = c[key]
	return
}

func (c testCycleState) Write(key string, val any) {
	c[key] = val
}

func (c testCycleState) Delete(key string) {
	delete(c, key)
}

var _ proto.Pod = &testPod{}

// testPod is test data just to set the nodeName
type testPod struct {
	nodeName string
}

func (t testPod) GetUid() string {
	return ""
}

func (t testPod) GetName() string {
	return ""
}

func (t testPod) GetNamespace() string {
	return ""
}

func (t testPod) GetApiVersion() string {
	return ""
}

func (t testPod) GetKind() string {
	return "pod"
}

func (t testPod) GetResourceVersion() string {
	return "v1"
}

func (t testPod) Spec() *protoapi.PodSpec {
	nodeName := t.nodeName
	return &protoapi.PodSpec{NodeName: &nodeName}
}

func (t testPod) Status() *protoapi.PodStatus {
	return nil
}
