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

// Package main is the entrypoint of the %.wasm file, compiled with
// '-target=wasi'. See /guest/RATIONALE.md for details.
package main

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/eventrecorder"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/plugin"
)

// main is compiled to a WebAssembly function named "_start", called by the
// wasm scheduler plugin during initialization.
func main() {
	p, err := New(klog.Get(), config.Get())
	if err != nil {
		panic(err)
	}
	plugin.Set(p)
}

func New(klog klogapi.Klog, jsonConfig []byte) (api.Plugin, error) {
	var args nodeNumberArgs
	if jsonConfig != nil {
		if err := json.Unmarshal(jsonConfig, &args); err != nil {
			panic(fmt.Errorf("decode arg into NodeNumberArgs: %w", err))
		}
		klog.Info("NodeNumberArgs is successfully applied")
	}
	return &NodeNumber{reverse: args.Reverse}, nil
}

// NodeNumber is an example plugin that favors nodes that share a numerical
// suffix with the pod name.
//
// For example, when a pod named "Pod1" is scheduled, a node named "Node1" gets
// a higher score than a node named "Node9".
//
// # Notes
//
//   - Only the last character in names are considered. This means "Node99" is
//     treated the same as "Node9"
//   - The reverse field inverts the score. For example, when `reverse == true`
//     a numeric match gets a results in a lower score than a match.
type NodeNumber struct {
	reverse bool
}

type nodeNumberArgs struct {
	Reverse bool `json:"reverse"`
}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name             = "NodeNumber"
	preScoreStateKey = "PreScore" + Name
)

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	podSuffixNumber uint8
}

// EventsToRegister implements api.EnqueueExtensions
func (pl *NodeNumber) EventsToRegister() []api.ClusterEvent {
	return []api.ClusterEvent{
		{Resource: api.Node, ActionType: api.Add},
	}
}

// PreScore implements api.PreScorePlugin
func (pl *NodeNumber) PreScore(state api.CycleState, pod proto.Pod, _ proto.NodeList) *api.Status {
	recorder := eventrecorder.Get()

	klog.InfoS("execute PreScore on NodeNumber plugin", "pod", klog.KObj(pod))

	podnum, ok := lastNumber(pod.Spec().GetNodeName())
	if !ok {
		recorder.Eventf(pod, nil, "PreScore", "not match lastNumber", "Skip", "")
		return nil // return success even if its suffix is non-number.
	}
	recorder.Eventf(pod, nil, "PreScore", "match lastNumber", "Continue", "")
	state.Write(preScoreStateKey, &preScoreState{podSuffixNumber: podnum})
	return nil
}

// Score implements api.ScorePlugin
func (pl *NodeNumber) Score(state api.CycleState, pod proto.Pod, nodeName string) (int32, *api.Status) {
	klog.InfoS("execute Score on NodeNumber plugin", "pod", klog.KObj(pod))

	var match bool
	if data, ok := state.Read(preScoreStateKey); ok {
		// Match is when there is a last digit, and it is the pod suffix.
		nodenum, ok := lastNumber(nodeName)
		match = ok && data.(*preScoreState).podSuffixNumber == nodenum
	} else {
		// Match is also when there is no pod spec node name.
		match = true
	}

	if pl.reverse {
		match = !match // invert the condition.
	}

	if match {
		return 10, nil
	}
	return 0, nil
}

// lastNumber returns the last number in the string or false.
func lastNumber(str string) (uint8, bool) {
	if len(str) == 0 {
		return 0, false
	}

	// We have at least a single character name. See if the last is a digit.
	lastChar := str[len(str)-1]
	if '0' <= lastChar && lastChar <= '9' {
		return lastChar - '0', true
	}
	return 0, false
}
