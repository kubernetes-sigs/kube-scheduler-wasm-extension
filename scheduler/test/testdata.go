package test

import (
	"bufio"
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
)

var URLErrorNotPlugin = localURL(pathWatError("not_plugin"))

var URLErrorPanicOnGetConfig = localURL(pathWatError("panic_on_get_config"))

var URLErrorPanicOnEnqueue = localURL(pathWatError("panic_on_enqueue"))

var URLErrorPanicOnPreFilter = localURL(pathWatError("panic_on_prefilter"))

var URLErrorPanicOnFilter = localURL(pathWatError("panic_on_filter"))

var URLErrorPanicOnPostFilter = localURL(pathWatError("panic_on_postfilter"))

var URLErrorPanicOnPreScore = localURL(pathWatError("panic_on_prescore"))

var URLErrorPreScoreWithoutScore = localURL(pathWatError("prescore_without_score"))

var URLErrorPanicOnScore = localURL(pathWatError("panic_on_score"))

var URLErrorPanicOnPreBind = localURL(pathWatError("panic_on_prebind"))

var URLErrorPanicOnBind = localURL(pathWatError("panic_on_bind"))

var URLErrorPanicOnStart = localURL(pathWatError("panic_on_start"))

var URLExampleNodeNumber = localURL(pathTinyGoExample("nodenumber"))

var URLExampleAdvanced = localURL(pathTinyGoExample("advanced"))

var URLTestAllNoopWat = localURL(pathWatTest("all_noop"))

var URLTestCycleState = localURL(pathTinyGoTest("cyclestate"))

var URLTestPreFilterFromGlobal = localURL(pathWatTest("prefilter_from_global"))

var URLTestFilter = localURL(pathTinyGoTest("filter"))

var URLTestFilterFromGlobal = localURL(pathWatTest("filter_from_global"))

var URLTestPostFilterFromGlobal = localURL(pathWatTest("postfilter_from_global"))

var URLTestPreScoreFromGlobal = localURL(pathWatTest("prescore_from_global"))

var URLTestScore = localURL(pathTinyGoTest("score"))

var URLTestScoreFromGlobal = localURL(pathWatTest("score_from_global"))

var URLTestPreBindFromGlobal = localURL(pathWatTest("prebind_from_global"))

var URLTestBindFromGlobal = localURL(pathWatTest("bind_from_global"))

var URLTestBind = localURL(pathTinyGoTest("bind"))

//go:embed testdata/yaml/node.yaml
var yamlNodeReal string

// NodeReal is a realistic v1.Node used for testing and benchmarks.
var NodeReal = func() *v1.Node {
	node := v1.Node{}
	decodeYaml(yamlNodeReal, &node)
	return &node
}()

var NodeSmallName = "good-node"

// NodeSmall is the smallest node that works with URLExampleFilterSimple.
var NodeSmall = &v1.Node{ObjectMeta: apimeta.ObjectMeta{Name: NodeSmallName}}

//go:embed testdata/yaml/pod.yaml
var yamlPodReal string

// PodReal is a realistic v1.Pod used for testing and benchmarks.
var PodReal = func() *v1.Pod {
	pod := v1.Pod{}
	decodeYaml(yamlPodReal, &pod)
	return &pod
}()

// PodSmall is the smallest pod that works with URLExampleFilterSimple.
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

// localURL prefixes file:// to a given path.
func localURL(path string) string {
	return "file://" + path
}

// pathTinyGoExample gets the absolute path to a given TinyGo example.
func pathTinyGoExample(name string) string {
	return relativeURL(path.Join("..", "..", "examples", name, "main.wasm"))
}

// pathTinyGoTest gets the absolute path to a given TinyGo test.
func pathTinyGoTest(name string) string {
	return relativeURL(path.Join("..", "..", "guest", "testdata", name, "main.wasm"))
}

// pathWatError gets the absolute path wasm compiled from a %.wat source.
func pathWatError(name string) string {
	return relativeURL(path.Join("testdata", "error", name+".wasm"))
}

// pathWatTest gets the absolute path wasm compiled from a %.wat source.
func pathWatTest(name string) string {
	return relativeURL(path.Join("testdata", "test", name+".wasm"))
}

// relativeURL gets the absolute from this file.
func relativeURL(fromThisFile string) string {
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
