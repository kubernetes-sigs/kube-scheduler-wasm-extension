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
	"fmt"

	guestapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prefilter"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// NodePorts is a plugin that checks if a node has free ports for the requested pod ports.
type NodePorts struct{}

func main() {
	plugin := &NodePorts{}

	prefilter.SetPlugin(plugin)
	filter.SetPlugin(plugin)
}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	name = "NodePorts"

	// preFilterStateKey is the key in CycleState to NodePorts pre-computed data.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + name

	// ErrReason when node ports aren't available.
	errReason = "node(s) didn't have free ports for the requested pod ports"

	// DefaultBindAllHostIP defines the default ip address used to bind to all host.
	defaultBindAllHostIP = "0.0.0.0"

	// ProtocolTCP is the TCP protocol.
	protocolTCP = "TCP"
)

type preFilterState []*protoapi.ContainerPort

// getContainerPorts returns the used host ports of Pods: if 'port' was used, a 'port:true' pair
// will be in the result; but it does not resolve port conflict.
func getContainerPorts(pods ...proto.Pod) []*protoapi.ContainerPort {
	ports := []*protoapi.ContainerPort{}
	for _, pod := range pods {
		for j := range pod.Spec().Containers {
			container := pod.Spec().Containers[j]
			for k := range container.Ports {
				if container.Ports[k].HostPort != nil && *container.Ports[k].HostPort <= 0 {
					continue
				}
				ports = append(ports, container.Ports[k])
			}
		}
	}
	return ports
}

// PreFilter invoked at the prefilter extension point.
func (pl *NodePorts) PreFilter(state guestapi.CycleState, pod proto.Pod) ([]string, *guestapi.Status) {
	s := getContainerPorts(pod)

	// Skip if a pod has no ports.
	if len(s) == 0 {
		return nil, &guestapi.Status{Code: guestapi.StatusCodeSkip}
	}

	state.Write(preFilterStateKey, preFilterState(s))
	return nil, nil
}

func getPreFilterState(state guestapi.CycleState) (preFilterState, error) {
	c, ok := state.Read(preFilterStateKey)
	if !ok {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("reading %q from cycleState error", preFilterStateKey)
	}

	s, ok := c.(preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to nodeports.preFilterState error", c)
	}
	return s, nil
}

func fitsPorts(wantPorts []*protoapi.ContainerPort, nodeInfo guestapi.NodeInfo) bool {
	// try to see whether existingPorts and wantPorts will conflict or not
	existingPorts := nodeInfo.UsedPorts()
	for _, cp := range wantPorts {

		// if existingPorts.CheckConfrict(cp.HostIP, string(cp.Protocol), cp.HostPort) {
		if checkConfrict(existingPorts, *cp.HostIP, *cp.Protocol, *cp.HostPort) {
			return false
		}
	}
	return true
}

// CheckConflict checks if the input (ip, protocol, port) conflicts with the existing
// ones in HostPortInfo.
func checkConfrict(h guestapi.HostPortInfo, ip, protocol string, port int32) bool {
	if port <= 0 {
		return false
	}

	sanitize(&ip, &protocol)

	pp := newProtocolPort(protocol, port)

	// If ip is 0.0.0.0 check all IP's (protocol, port) pair
	if ip == defaultBindAllHostIP {
		for _, m := range h {
			if _, ok := m[*pp]; ok {
				return true
			}
		}
		return false
	}

	// If ip isn't 0.0.0.0, only check IP and 0.0.0.0's (protocol, port) pair
	for _, key := range []string{defaultBindAllHostIP, ip} {
		if m, ok := h[key]; ok {
			if _, ok2 := m[*pp]; ok2 {
				return true
			}
		}
	}

	return false
}

// sanitize the parameters
func sanitize(ip, protocol *string) {
	if len(*ip) == 0 {
		*ip = defaultBindAllHostIP
	}
	if len(*protocol) == 0 {
		*protocol = protocolTCP
	}
}

// newProtocolPort creates a ProtocolPort instance.
func newProtocolPort(protocol string, port int32) *guestapi.ProtocolPort {
	pp := &guestapi.ProtocolPort{
		Protocol: protocol,
		Port:     port,
	}

	if len(pp.Protocol) == 0 {
		pp.Protocol = string(protocolTCP)
	}

	return pp
}

// Filter invoked at the filter extension point.
func (pl *NodePorts) Filter(state guestapi.CycleState, pod proto.Pod, nodeInfo guestapi.NodeInfo) *guestapi.Status {
	wantPorts, err := getPreFilterState(state)
	if err != nil {
		return &guestapi.Status{Code: guestapi.StatusCodeError, Reason: "failed to get PreFilterState"}
	}

	fits := fitsPorts(wantPorts, nodeInfo)
	if fits {
		return &guestapi.Status{Code: guestapi.StatusCodeUnschedulable, Reason: errReason}
	}
	return nil
}
