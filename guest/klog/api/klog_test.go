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

package api_test

import (
	"fmt"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
)

var (
	klog api.Klog = api.UnimplementedKlog{}
	pod  proto.Pod
)

func ExampleKlog() {
	klog.Info("NodeNumberArgs is successfully applied")

	// For structured logging, use pairs, wrapping values in api.KObj where possible.
	klog.InfoS("execute Score on NodeNumber plugin", "pod", api.KObj(pod))

	metricName := "scheduler_framework_extension_point_duration_milliseconds"
	err := fmt.Errorf("metric %q not found", metricName)
	klog.Error(err)

	bucketSize := 32
	histBucketSize := 16
	index := 2
	err = fmt.Errorf("found different bucket size: expect %v, but got %v at index %v", bucketSize, histBucketSize, index)
	labels := map[string]string{"Name": "some-name"}
	klog.ErrorS(err, "the validation for HistogramVec is failed. The data for this metric won't be stored in a benchmark result file", "metric", metricName, "labels", labels)

	// Output:
	//
}
