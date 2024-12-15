//go:build !wasm

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

// Package klog_test ensures that even though this package can't be tested with
// `tinygo test -target=wasi`, due to imports required, it can be tested with
// normal Go (due to stubbed implementation).
package klog_test

import (
	"fmt"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
)

func ExampleInfo() {
	klog.Info("NodeNumberArgs is successfully applied")

	// Output:
	//
}

var pod proto.Pod

func ExampleInfoS() {
	klog.InfoS("execute Score on NodeNumber plugin", "pod", klogapi.KObj(pod))

	// Output:
	//
}

func ExampleError() {
	metricName := "scheduler_framework_extension_point_duration_milliseconds"
	err := fmt.Errorf("metric %q not found", metricName)
	klog.Error(err)

	// Output:
	//
}

func ExampleErrorS() {
	bucketSize := 32
	histBucketSize := 16
	index := 2
	err := fmt.Errorf("found different bucket size: expect %v, but got %v at index %v", bucketSize, histBucketSize, index)
	metricName := "scheduler_framework_extension_point_duration_milliseconds"
	labels := map[string]string{"Name": "some-name"}

	klog.ErrorS(err, "the validation for HistogramVec is failed. The data for this metric won't be stored in a benchmark result file", "metric", metricName, "labels", labels)

	// Output:
	//
}
