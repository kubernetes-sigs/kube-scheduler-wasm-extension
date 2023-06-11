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

var PathErrorNotPlugin = pathWatError("not_plugin")

var PathErrorPanicOnFilter = pathWatError("panic_on_filter")

var PathErrorPanicOnScore = pathWatError("panic_on_score")

var PathErrorPanicOnStart = pathWatError("panic_on_start")

var PathExampleFilterSimple = pathTinyGoExample("filter-simple")

var PathExampleScoreSimple = pathTinyGoExample("score-simple")

var PathTestFilterFromGlobal = pathWatTest("filter_from_global")

var PathTestNoopTinyGo = pathTinyGoTest("noop")

var PathTestNoopWat = pathWatTest("noop")

var PathTestScoreFromGlobal = pathWatTest("score_from_global")

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

// pathTinyGoExample gets the absolute path to a given TinyGo example.
func pathTinyGoExample(name string) string {
	return relativePath(path.Join("..", "..", "examples", name, "main.wasm"))
}

// pathTinyGoTest gets the absolute path to a given TinyGo test.
func pathTinyGoTest(name string) string {
	return relativePath(path.Join("..", "..", "guest", "testdata", name, "main.wasm"))
}

// pathWatError gets the absolute path wasm compiled from a %.wat source.
func pathWatError(name string) string {
	return relativePath(path.Join("testdata", "error", name+".wasm"))
}

// pathWatTest gets the absolute path wasm compiled from a %.wat source.
func pathWatTest(name string) string {
	return relativePath(path.Join("testdata", "test", name+".wasm"))
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
	} else {
		return abs
	}
}
