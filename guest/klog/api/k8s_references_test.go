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

package api

import (
	"fmt"
	"reflect"
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
)

var _ proto.Metadata = &testMetadata{}

type testMetadata struct {
	Name, Namespace, UID string
}

func (t *testMetadata) GetName() string {
	return t.Name
}

func (t *testMetadata) GetNamespace() string {
	return t.Namespace
}

func (t *testMetadata) GetUid() string {
	return t.UID
}

var _ proto.Metadata = &panicMetadata{}

type panicMetadata struct {
	Name, Namespace, UID string
}

func (panicMetadata) GetName() string {
	panic("unexpected")
}

func (panicMetadata) GetNamespace() string {
	panic("unexpected")
}

func (panicMetadata) GetUid() string {
	panic("unexpected")
}

func TestKObj(t *testing.T) {
	tests := []struct {
		name     string
		input    proto.Metadata
		expected *objectRef
	}{
		{
			name: "nil",
		},
		{
			// If calling KObj called methods, it would hurt performance as
			// unmarshalling is very expensive in wasm.
			name:  "doesn't call methods",
			input: panicMetadata{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.input, KObj(tc.input).(*objectRef).Metadata; want != have {
				t.Fatalf("unexpected ref: %v != %v", want, have)
			}
		})
	}
}

func TestKObjSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []proto.Metadata
		expected *objectRef
	}{
		{
			name: "nil",
		},
		{
			// If calling KObjSlice called methods, it would hurt performance
			// as unmarshalling is very expensive in wasm.
			name:  "doesn't call methods",
			input: []proto.Metadata{panicMetadata{}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.input, KObjSlice(tc.input).(*kobjSlice[proto.Metadata]).objs; !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected refs: %v != %v", want, have)
			}
		})
	}
}

func TestKObjSliceFn(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		have := KObjSliceFn((func() []proto.Metadata)(nil))
		if have.(*kObjSliceFn[proto.Metadata]).fn != nil {
			t.Fatalf("unexpected fn: %v", have)
		}
		if want, have := "[]", have.String(); want != have {
			t.Fatalf("unexpected string: %v != %v", want, have)
		}
	})

	// If creating a KObjSliceFn called Get, it would hurt performance as
	// unmarshalling is very expensive in wasm.
	t.Run("doesn't call methods", func(t *testing.T) {
		lazySlice := panicLazySlice{}
		have := KObjSliceFn(lazySlice.Get)
		if lazySlice.called == true {
			t.Fatalf("unexpected call to items")
		}
		if want, have := "[good-pod]", have.String(); want != have {
			t.Fatalf("unexpected string: %v != %v", want, have)
		}
	})
}

type panicLazySlice struct{ called bool }

func (panicLazySlice) Get() []proto.Metadata {
	return []proto.Metadata{&testMetadata{Name: "good-pod"}}
}

func TestKObj_String(t *testing.T) {
	tests := []struct {
		name     string
		input    fmt.Stringer
		expected string
	}{
		{
			name:  "nil -> empty",
			input: KObj(nil),
		},
		{
			name:  "empty -> empty",
			input: KObj(&testMetadata{}),
		},
		{
			name:     "name but not ns",
			input:    KObj(&testMetadata{Name: "good-pod"}),
			expected: "good-pod",
		},
		{
			name:     "ns but not name",
			input:    KObj(&testMetadata{Namespace: "test"}),
			expected: "test/",
		},
		{
			name: "all",
			input: KObj(&testMetadata{
				Name:      "good-pod",
				Namespace: "test",
				UID:       "384900cd-dc7b-41ec-837e-9c4c1762363e",
			}),
			expected: "test/good-pod",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.expected, tc.input.String(); want != have {
				t.Fatalf("unexpected string: %v != %v", want, have)
			}
		})
	}
}

func Test_sliceString(t *testing.T) {
	tests := []struct {
		name     string
		input    []proto.Metadata
		expected string
	}{
		{
			name:     "nil -> empty slice",
			input:    nil,
			expected: "[]",
		},
		{
			name:     "empty -> empty slice",
			input:    []proto.Metadata{&testMetadata{}},
			expected: "[]",
		},
		{
			name:     "name but not ns",
			input:    []proto.Metadata{&testMetadata{Name: "good-pod"}},
			expected: "[good-pod]",
		},
		{
			name:     "ns but not name",
			input:    []proto.Metadata{&testMetadata{Namespace: "test"}},
			expected: "[test/]",
		},
		{
			name: "all",
			input: []proto.Metadata{
				&testMetadata{
					Name:      "good-pod",
					Namespace: "test",
					UID:       "384900cd-dc7b-41ec-837e-9c4c1762363e",
				},
			},
			expected: "[test/good-pod]",
		},
		{
			name: "multiple",
			input: []proto.Metadata{
				&testMetadata{
					Name:      "good-pod",
					Namespace: "test",
					UID:       "384900cd-dc7b-41ec-837e-9c4c1762363e",
				},
				&testMetadata{Name: "bad-pod"},
			},
			expected: "[test/good-pod bad-pod]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.expected, kobjSliceString(tc.input); want != have {
				t.Fatalf("unexpected string: %v != %v", want, have)
			}
		})
	}
}
