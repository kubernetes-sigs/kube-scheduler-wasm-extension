package test

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	v1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework"

	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
)

// NewPluginExampleFilterSimple returns a new plugin configured with PathExampleFilterSimple.
func NewPluginExampleFilterSimple(ctx context.Context) (frameworkruntime.Plugin, error) {
	return wasm.NewFromConfig(ctx, wasm.PluginConfig{
		GuestName: "filter-simple",
		GuestPath: PathExampleFilterSimple,
	})
}

var PathErrorPanicOnFilter = pathError("panic_on_filter")

var PathTestPanicOnStart = pathError("panic_on_start")

var PathExampleFilterSimple = pathExample("filter-simple")

// NewPluginExampleNoop returns a new plugin configured with PathExampleNoop.
func NewPluginExampleNoop(ctx context.Context) (frameworkruntime.Plugin, error) {
	return wasm.NewFromConfig(ctx, wasm.PluginConfig{
		GuestName: "noop",
		GuestPath: PathExampleNoop,
	})
}

var PathExampleNoop = pathExample("noop")

//go:embed testdata/yaml/node.yaml
var yamlNodeReal string

// NodeReal is a realistic v1.Node used for testing and benchmarks.
var NodeReal = func() *v1.Node {
	node := v1.Node{}
	decodeYaml(yamlNodeReal, &node)
	return &node
}()

// NodeSmall is the smallest node that works with PathExampleFilterSimple.
var NodeSmall = &v1.Node{ObjectMeta: apimeta.ObjectMeta{Name: "good-node"}}

//go:embed testdata/yaml/pod.yaml
var yamlPodReal string

// PodReal is a realistic v1.Pod used for testing and benchmarks.
var PodReal = func() *v1.Pod {
	pod := v1.Pod{}
	decodeYaml(yamlPodReal, &pod)
	return &pod
}()

// PodSmall is the smallest pod that works with PathExampleFilterSimple.
var PodSmall = &v1.Pod{
	ObjectMeta: apimeta.ObjectMeta{
		Name:      "good-pod",
		Namespace: "test",
		UID:       "384900cd-dc7b-41ec-837e-9c4c1762363e",
	},
	Spec: v1.PodSpec{NodeName: NodeSmall.Name},
}

func decodeYaml[O apiruntime.Object](yaml string, object O) {
	reader := bufio.NewReader(strings.NewReader(yaml))
	r := apiyaml.NewYAMLReader(reader)
	doc, err := r.Read()
	if err != nil {
		panic(fmt.Errorf("could not read yaml: %w", err))
	}

	d := scheme.Codecs.UniversalDeserializer()
	_, _, err = d.Decode(doc, nil, object)
	if err != nil {
		panic(fmt.Errorf("could not decode yaml: %w", err))
	}
}

// pathExample gets the absolute path to a given example.
func pathExample(name string) string {
	return relativePath(path.Join("..", "..", "examples", name, "main.wasm"))
}

// pathError gets the absolute path wasm compiled from a %.wat source.
func pathError(name string) string {
	return relativePath(path.Join("testdata", "error", name+".wasm"))
}

// relativePath gets the absolute from this file.
func relativePath(fromThisFile string) string {
	_, thisFile, _, ok := runtime.Caller(1)
	if !ok {
		panic("cannot determine current path")
	}
	p := path.Join(path.Dir(thisFile), fromThisFile)
	if abs, err := filepath.Abs(p); err != nil {
		panic(err)
		return ""
	} else {
		return abs
	}
}
