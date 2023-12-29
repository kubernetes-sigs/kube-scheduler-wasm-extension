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

package guest_test

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	"fmt"
	"testing"

	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

var nodeSmall = &protoapi.Node{Metadata: &meta.ObjectMeta{Name: stringToPointer("good-node")}}

var podSmall = &protoapi.Pod{
	Metadata: &meta.ObjectMeta{
		Name:      stringToPointer("good-pod"),
		Namespace: stringToPointer("test"),
		Uid:       stringToPointer("384900cd-dc7b-41ec-837e-9c4c1762363e"),
	},
	Spec: &protoapi.PodSpec{NodeName: nodeSmall.Metadata.Name},
}

var _ proto.Metadata = &testMetadata{}

type testMetadata struct {
	Name, Namespace, UID, ResourceVersion string
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

func (t *testMetadata) GetResourceVersion() string {
	return t.ResourceVersion
}

type stringerFunc func() string

func (s stringerFunc) String() string {
	return s()
}

// BenchmarkKlog shows that slice functions like api.KObjSlice are more optimal
// than doing concatenation manually.
func BenchmarkKlog(b *testing.B) {
	pod := &testMetadata{
		Name:      "good-pod",
		Namespace: "test",
		UID:       "384900cd-dc7b-41ec-837e-9c4c1762363e",
	}

	benches := []struct {
		name  string
		input fmt.Stringer
	}{
		{
			name: "KObj",
			input: stringerFunc(func() string {
				return fmt.Sprint("[", klogapi.KObj(pod), klogapi.KObj(pod), "]")
			}),
		},
		{
			name:  "KObjSlice",
			input: klogapi.KObjSlice([]proto.Metadata{pod, pod}),
		},
		{
			name: "KObjSliceFn",
			input: klogapi.KObjSliceFn(func() []proto.Metadata {
				return []proto.Metadata{pod, pod}
			}),
		},
	}

	for _, bc := range benches {
		b.Run(bc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = bc.input.String()
			}
		})
	}
}

func BenchmarkUnmarshalVT(b *testing.B) {
	unmarshalNode := func(data []byte) error {
		var msg protoapi.Node
		return msg.UnmarshalVT(data)
	}

	unmarshalPod := func(data []byte) error {
		var msg protoapi.Pod
		return msg.UnmarshalVT(data)
	}

	// TODO: Find a way to convert yaml to proto in a way that compiles in
	// TinyGo, so that we can use real data. Or check in the serialized protos.
	benches := []struct {
		name      string
		input     []byte
		unmarshal func(data []byte) error
	}{
		{
			name:      "node: small",
			input:     mustMarshal(b, nodeSmall.MarshalVT),
			unmarshal: unmarshalNode,
		},
		{
			name:      "pod: small",
			input:     mustMarshal(b, podSmall.MarshalVT),
			unmarshal: unmarshalPod,
		},
	}

	for _, bc := range benches {
		b.Run(bc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := bc.unmarshal(bc.input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func stringToPointer(s string) *string {
	return &s
}

func mustMarshal(b *testing.B, marshal func() (data []byte, err error)) []byte {
	proto, err := marshal()
	if err != nil {
		b.Fatal(err)
	}
	return proto
}
