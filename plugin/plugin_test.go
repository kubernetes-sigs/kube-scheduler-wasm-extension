package plugin

import (
	"context"
	"io"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
)

func TestFilterABI(t *testing.T) {
	testFilter(t, "testdata/abimain/main.wasm")
}

func TestFilterVFS(t *testing.T) {
	testFilter(t, "testdata/vfsmain/main.wasm")
}

func testFilter(t *testing.T, guestPath string) {
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
		{
			name: "filtered: bad-node",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "good-node",
				},
			},
			node:         st.MakeNode().Name("bad-node").Obj(),
			expectedCode: framework.Unschedulable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := New(guestPath)
			if err != nil {
				t.Fatalf("failed to create plugin: %v", err)
			}
			defer p.(io.Closer).Close()

			ni := framework.NewNodeInfo()
			ni.SetNode(test.node)
			s := p.(framework.FilterPlugin).Filter(context.Background(), nil, test.pod, ni)
			if s.Code() != test.expectedCode {
				t.Fatalf("unexpected code: got %v, expected %v, got reason: %v", s.Code(), test.expectedCode, s.Message())
			}
		})
	}
}
