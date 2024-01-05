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

package scheduler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	k8stest "k8s.io/klog/v2/test"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"sigs.k8s.io/kube-scheduler-wasm-extension/internal/e2e"
	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

// TestCycleStateCoherence ensures cycle state data is coherent in a scheduling
// context.
func TestCycleStateCoherence(t *testing.T) {
	ctx := context.Background()

	plugin, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{GuestURL: test.URLTestCycleState}, nil)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer plugin.(io.Closer).Close()

	pod := test.PodReal
	ni := framework.NewNodeInfo()
	ni.SetNode(test.NodeReal)

	// run: the guest will crash if any of the callbacks see a different pod.
	e2e.RunAll(ctx, t, plugin, pod, ni)
	// run again: the guest will crash if it sees the same pointer.
	e2e.RunAll(ctx, t, plugin, pod, ni)
}

func TestExample_NodeNumber(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		testExample_NodeNumber(t, false)
	})
	t.Run("Advanced", func(t *testing.T) {
		testExample_NodeNumber(t, true)
	})
}

func testExample_NodeNumber(t *testing.T, advanced bool) {
	// Reinit klog for tests.
	fs := k8stest.InitKlog(t)

	// Disable timestamps.
	fs.Set("skip_headers", "true") //nolint

	ctx := context.Background()
	plugin := newNodeNumberPlugin(ctx, t, advanced, false, 0)
	defer plugin.(io.Closer).Close()

	pod := &v1.Pod{ObjectMeta: v1meta.ObjectMeta{Name: "happy8-meta"}, Spec: v1.PodSpec{NodeName: "happy8"}}

	t.Run("Score zero on unmatch", func(t *testing.T) {
		// The pod spec node name doesn't end with the same number as the node, so
		// we expect to score zero.
		var buf bytes.Buffer
		klog.SetOutput(&buf)

		score := e2e.RunAll(ctx, t, plugin, pod, nodeInfoWithName("glad9"))
		if want, have := int64(0), score; want != have {
			t.Fatalf("unexpected score: want %v, have %v", want, have)
		}

		want := `"execute PreScore on NodeNumber plugin" pod="happy8-meta"
"execute Score on NodeNumber plugin" pod="happy8-meta"
` // klog always adds newline
		if have := buf.String(); want != have {
			t.Fatalf("unexpected log: want %v, have %v", want, have)
		}
	})

	t.Run("Score ten on match", func(t *testing.T) {
		// The pod spec node name isn't the same as the node name. However,
		// they both end in the same number, so we expect to score ten.
		var buf bytes.Buffer
		klog.SetOutput(&buf)

		score := e2e.RunAll(ctx, t, plugin, pod, nodeInfoWithName("glad8"))
		if want, have := int64(10), score; want != have {
			t.Fatalf("unexpected score: want %v, have %v", want, have)
		}

		wantLog := `"execute PreScore on NodeNumber plugin" pod="happy8-meta"
"execute Score on NodeNumber plugin" pod="happy8-meta"`
		if wantLog != strings.TrimSpace(buf.String()) {
			t.Fatalf("unexpected log: want %v, have %v", wantLog, buf.String())
		}
	})

	t.Run("Reverse means score zero on match", func(t *testing.T) {
		// This proves we can read configuration.
		var buf bytes.Buffer
		klog.SetOutput(&buf)

		reversed := newNodeNumberPlugin(ctx, t, advanced, true, 0)
		defer reversed.(io.Closer).Close()

		score := e2e.RunAll(ctx, t, reversed, pod, nodeInfoWithName("glad8"))
		if want, have := int64(0), score; want != have {
			t.Fatalf("unexpected score: want %v, have %v", want, have)
		}

		wantLog := `NodeNumberArgs is successfully applied
"execute PreScore on NodeNumber plugin" pod="happy8-meta"
"execute Score on NodeNumber plugin" pod="happy8-meta"`
		if wantLog != strings.TrimSpace(buf.String()) {
			t.Fatalf("unexpected log: want %v, have %v", wantLog, buf.String())
		}
	})
}

func BenchmarkExample_NodeNumber(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		benchmarkExample_NodeNumber(b, false, 3)
	})
	b.Run("Simple Log", func(b *testing.B) {
		benchmarkExample_NodeNumber(b, false, 0)
	})
	b.Run("Advanced", func(b *testing.B) {
		benchmarkExample_NodeNumber(b, true, 3)
	})
	b.Run("Advanced Log", func(b *testing.B) {
		benchmarkExample_NodeNumber(b, true, 0)
	})
}

func benchmarkExample_NodeNumber(b *testing.B, advanced bool, logSeverity int32) {
	b.Helper()
	// Reinit klog for tests.
	fs := k8stest.InitKlog(b)
	// Disable timestamps.
	fs.Set("skip_headers", "true") //nolint

	klog.SetOutput(io.Discard)

	ctx := context.Background()

	b.Run("New", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			newNodeNumberPlugin(ctx, b, advanced, false, logSeverity).(io.Closer).Close()
		}
	})

	plugin := newNodeNumberPlugin(ctx, b, advanced, false, logSeverity)
	defer plugin.(io.Closer).Close()

	pod := *test.PodReal // copy
	pod.Spec.NodeName = "happy8"

	b.Run("Run", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			score := e2e.RunAll(ctx, b, plugin, &pod, nodeInfoWithName("glad8"))
			if want, have := int64(10), score; want != have {
				b.Fatalf("unexpected score: want %v, have %v", want, have)
			}
		}
	})
}

func newNodeNumberPlugin(ctx context.Context, t e2e.Testing, advanced, reverse bool, logSeverity int32) framework.Plugin {
	t.Helper()
	guestURL := test.URLExampleNodeNumber
	if advanced {
		guestURL = test.URLExampleAdvanced
	}
	plugin, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{
		GuestURL:    guestURL,
		LogSeverity: logSeverity,
		GuestConfig: fmt.Sprintf(`{"reverse": %v}`, reverse),
	}, nil)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	return plugin
}

func nodeInfoWithName(name string) *framework.NodeInfo {
	ni := framework.NewNodeInfo()
	node := *test.NodeReal // copy
	node.ObjectMeta.Name = name
	ni.SetNode(&node)
	return ni
}
