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

package main

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

func main() {
	guest.Filter = api.FilterFunc(nameEqualsPodSpec)
}

// nameEqualsPodSpec schedules this node if its name equals its pod spec.
func nameEqualsPodSpec(args api.FilterArgs) (api.StatusCode, string) {
	nodeName := nilToEmpty(args.NodeInfo().Node().Metadata.Name)
	podSpecNodeName := nilToEmpty(args.Pod().Spec.NodeName)

	if len(podSpecNodeName) == 0 || podSpecNodeName == nodeName {
		return api.StatusCodeSuccess, ""
	} else {
		return api.StatusCodeUnschedulable, podSpecNodeName + " != " + nodeName
	}
}

func nilToEmpty(ptr *string) (s string) {
	if ptr != nil {
		s = *ptr
	}
	return
}
