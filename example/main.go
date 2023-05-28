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
	"context"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/framework"
)

func main() {
	guest.RegisterFilter(&NodeName{})
}

// NodeName is a plugin that checks if a pod spec node name matches the current node.
type NodeName struct{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	name = "NodeName"

	// ErrReason returned when node name doesn't match.
	ErrReason = "node(s) didn't match the requested node name"
)

func (*NodeName) Name() string {
	return name
}

func (*NodeName) Filter(ctx context.Context, state *framework.CycleState, podFn framework.PodFn, nodeInfoFn framework.NodeInfoFn) *framework.Status {
	nodeName := nilToEmpty(nodeInfoFn().Node().Metadata.Name)
	podSpecNodeName := nilToEmpty(podFn().Spec.NodeName)

	if len(podSpecNodeName) == 0 || podSpecNodeName == nodeName {
		return nil
	} else {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReason)
	}
}

func nilToEmpty(ptr *string) (s string) {
	if ptr != nil {
		s = *ptr
	}
	return
}
