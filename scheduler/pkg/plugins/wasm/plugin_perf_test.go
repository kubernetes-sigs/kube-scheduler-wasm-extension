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

package wasm

import (
	"context"
	"io"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
)

func BenchmarkFilter(b *testing.B) {
	tests := []struct {
		name         string
		pod          *v1.Pod
		node         *v1.Node
		expectedCode framework.Code
	}{
		{
			name: "success: node is match with spec.NodeName",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "good-node",
				},
			},
			node:         st.MakeNode().Name("good-node").Obj(),
			expectedCode: framework.Success,
		},
	}

	p, err := New("../example/main.wasm")
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}
	defer p.(io.Closer).Close()

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			ni := framework.NewNodeInfo()
			ni.SetNode(test.node)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s := p.(framework.FilterPlugin).Filter(context.Background(), nil, test.pod, ni)
				if s.Code() != test.expectedCode {
					b.Fatalf("unexpected code: got %v, expected %v, got reason: %v", s.Code(), test.expectedCode, s.Message())
				}
			}
		})
	}
}
