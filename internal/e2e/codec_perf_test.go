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
	"testing"

	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

// BenchmarkUnmarshalVT helps explain protobuf unmarshalling performance. Even
// though this doesn't run in Wasm, improvements here likely help TinyGo.
func BenchmarkUnmarshalVT(b *testing.B) {
	unmarshalNode := func(data []byte) error {
		var msg protoapi.Node
		return msg.UnmarshalVT(data)
	}

	unmarshalPod := func(data []byte) error {
		var msg protoapi.Pod
		return msg.UnmarshalVT(data)
	}

	tests := []struct {
		name      string
		input     []byte
		unmarshal func(data []byte) error
	}{
		{
			name:      "node: small",
			input:     mustMarshal(b, test.NodeSmall.Marshal),
			unmarshal: unmarshalNode,
		},
		{
			name:      "node: real",
			input:     mustMarshal(b, test.NodeReal.Marshal),
			unmarshal: unmarshalNode,
		},
		{
			name:      "pod: small",
			input:     mustMarshal(b, test.PodSmall.Marshal),
			unmarshal: unmarshalPod,
		},
		{
			name:      "pod: real",
			input:     mustMarshal(b, test.PodReal.Marshal),
			unmarshal: unmarshalPod,
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := tc.unmarshal(tc.input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func mustMarshal(b *testing.B, marshal func() (data []byte, err error)) []byte {
	proto, err := marshal()
	if err != nil {
		b.Fatal(err)
	}
	return proto
}
