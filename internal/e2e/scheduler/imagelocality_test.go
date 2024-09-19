package scheduler_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"sigs.k8s.io/kube-scheduler-wasm-extension/internal/e2e"
	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

func Test_ImageLocality(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := []*v1.Node{
		{
			ObjectMeta: v1meta.ObjectMeta{Name: "no-image-node1"},
			Status: v1.NodeStatus{
				Images: []v1.ContainerImage{
					{Names: []string{"unrelated-image:v1"}, SizeBytes: 100000000},
					{Names: []string{"unrelated-image:v2"}, SizeBytes: 100000000},
					{Names: []string{"unrelated-image:v3"}, SizeBytes: 100000000},
				},
			},
		},
		{
			ObjectMeta: v1meta.ObjectMeta{Name: "image-node1"},
			Status: v1.NodeStatus{
				Images: []v1.ContainerImage{
					{Names: []string{"test-image:v1"}, SizeBytes: 100000000},
					{Names: []string{"test-image:v2"}, SizeBytes: 100000000},
					{Names: []string{"test-image:v3"}, SizeBytes: 100000000},
				},
			},
		},
		{
			ObjectMeta: v1meta.ObjectMeta{Name: "image-node2"},
			Status: v1.NodeStatus{
				Images: []v1.ContainerImage{
					{Names: []string{"test-image:v1"}, SizeBytes: 100000000},
					{Names: []string{"test-image:v2"}, SizeBytes: 100000000},
					{Names: []string{"test-image:v3"}, SizeBytes: 100000000},
				},
			},
		},
	}
	imageExistenceMap := createImageExistenceMap(nodes)

	ninfos := make([]*framework.NodeInfo, len(nodes))
	for i, node := range nodes {
		ninfos[i] = framework.NewNodeInfo()
		ninfos[i].SetNode(node)
		ninfos[i].ImageStates = getNodeImageStates(node, imageExistenceMap)
	}

	handle := &test.FakeHandle{
		SharedLister: &test.FakeSharedLister{
			NodeInfoLister: &test.FakeNodeInfoLister{
				Nodes: ninfos,
			},
		},
	}

	plugin, err := wasm.NewFromConfig(ctx, "wasm", wasm.PluginConfig{
		GuestURL:    test.URLExampleImageLocality,
		LogSeverity: 0,
	}, handle)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}
	defer plugin.(io.Closer).Close()

	t.Run("unmatch", func(t *testing.T) {
		pod := &v1.Pod{ObjectMeta: v1meta.ObjectMeta{Name: "happy8-meta"}, Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "happy8",
					Image: "test-image:v1",
				},
			},
		}}

		var buf bytes.Buffer
		klog.SetOutput(&buf)

		// ninfos[0] is a node that doesn't have the requested image.
		// so we expect to score zero.
		score := e2e.RunAll(ctx, t, plugin, pod, ninfos[0], nil)
		if want, have := int64(0), score; want != have {
			t.Fatalf("unexpected score: want %v, have %v", want, have)
		}
	})
	t.Run("match", func(t *testing.T) {
		pod := &v1.Pod{ObjectMeta: v1meta.ObjectMeta{Name: "happy8-meta"}, Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "happy8",
					Image: "test-image:v1",
				},
			},
		}}

		var buf bytes.Buffer
		klog.SetOutput(&buf)

		// ninfos[1] and ninfos[2] are nodes that have the requested image.
		// so we expect to score non-zero.
		score := e2e.RunAll(ctx, t, plugin, pod, ninfos[1], nil)
		if want, have := int64(4), score; want != have {
			t.Fatalf("unexpected score: want %v, have %v", want, have)
		}
	})
}

// getNodeImageStates returns the given node's image states based on the given imageExistence map.
func getNodeImageStates(node *v1.Node, imageExistenceMap map[string]sets.Set[string]) map[string]*framework.ImageStateSummary {
	imageStates := make(map[string]*framework.ImageStateSummary)

	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			imageStates[name] = &framework.ImageStateSummary{
				Size:     image.SizeBytes,
				NumNodes: imageExistenceMap[name].Len(),
			}
		}
	}
	return imageStates
}

// createImageExistenceMap returns a map recording on which nodes the images exist, keyed by the images' names.
func createImageExistenceMap(nodes []*v1.Node) map[string]sets.Set[string] {
	imageExistenceMap := make(map[string]sets.Set[string])
	for _, node := range nodes {
		for _, image := range node.Status.Images {
			for _, name := range image.Names {
				if _, ok := imageExistenceMap[name]; !ok {
					imageExistenceMap[name] = sets.New(node.Name)
				} else {
					imageExistenceMap[name].Insert(node.Name)
				}
			}
		}
	}
	return imageExistenceMap
}
