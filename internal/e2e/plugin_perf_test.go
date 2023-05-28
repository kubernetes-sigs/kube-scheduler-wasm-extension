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
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/testdata"
)

func BenchmarkPluginFilter(b *testing.B) {
	noop, err := testdata.NewPluginExampleNoop()
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}
	defer noop.(io.Closer).Close()

	filterSimple, err := testdata.NewPluginExampleFilterSimple()
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}
	defer filterSimple.(io.Closer).Close()

	plugins := []struct {
		name   string
		plugin framework.Plugin
	}{
		{
			name:   "noop",
			plugin: noop,
		},
		{
			name:   "filter-simple",
			plugin: filterSimple,
		},
	}

	tests := []struct {
		name string
		node *v1.Node
		pod  *v1.Pod
	}{
		{
			name: "params: small",
			node: testdata.NodeSmall,
			pod:  testdata.PodSmall,
		},
		{
			name: "params: real",
			node: testdata.NodeReal,
			pod:  testdata.PodReal,
		},
	}

	for _, tp := range plugins {
		pl := tp
		b.Run(pl.name, func(b *testing.B) {
			for _, tc := range tests {
				b.Run(tc.name, func(b *testing.B) {
					ni := framework.NewNodeInfo()
					ni.SetNode(tc.node)

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						s := pl.plugin.(framework.FilterPlugin).Filter(context.Background(), nil, tc.pod, ni)
						if want, have := framework.Success, s.Code(); want != have {
							b.Fatalf("unexpected code: got %v, expected %v, got reason: %v", want, have, s.Message())
						}
					}
				})
			}
		})
	}
}
