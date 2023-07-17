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

package wasm

import (
	"context"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	i32                             = wazeroapi.ValueTypeI32
	i64                             = wazeroapi.ValueTypeI64
	k8sApi                          = "k8s.io/api"
	k8sApiNodeInfoNode              = "nodeInfo/node"
	k8sApiNodeName                  = "nodeName"
	k8sApiPod                       = "pod"
	k8sScheduler                    = "k8s.io/scheduler"
	k8sSchedulerResultClusterEvents = "result.cluster_events"
	k8sSchedulerResultNodeNames     = "result.node_names"
	k8sSchedulerResultStatusReason  = "result.status_reason"
)

func instantiateHostApi(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder(k8sApi).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiNodeInfoNodeFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiNodeInfoNode).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiNodeNameFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiNodeName).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiPodFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiPod).
		Instantiate(ctx)
}

func instantiateHostScheduler(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder(k8sScheduler).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultClusterEventsFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultClusterEvents).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultNodeNamesFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultNodeNames).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultStatusReasonFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultStatusReason).
		Instantiate(ctx)
}

// stackKey is a context.Context value associated with a stack
// pointer to the current request.
type stackKey struct{}

// stack holds any parameters or results from functions implemented by the
// guest. An instance of stack is only used for a single function invocation,
// such as guest.filterFn.
//
// # Notes
//
//   - This is needed because WebAssembly types are numeric only.
//   - Result fields are conventionally prefixed with "result".
//   - Declaring one type is less complicated than one+context key per
//     function. Functions should ignore fields they don't use.
type stack struct {
	// pod is used by guest.filterFn and guest.scoreFn
	pod *v1.Pod

	// nodeInfo is used by guest.filterFn
	nodeInfo *framework.NodeInfo

	// nodeName is used by guest.scoreFn
	nodeName string

	// resultClusterEvents is returned by guest.enqueueFn
	resultClusterEvents []framework.ClusterEvent

	// resultNodeNames is returned by guest.prefilterFn
	resultNodeNames []string

	// reason returned by all guest exports except guest.enqueueFn
	//
	// It is a field to avoid compiler-specific malloc/free functions, and to
	// avoid having to deal with out-params because TinyGo only supports a
	// single result.
	resultStatusReason string
}

func paramsFromContext(ctx context.Context) *stack {
	return ctx.Value(stackKey{}).(*stack)
}

func k8sApiNodeInfoNodeFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	node := paramsFromContext(ctx).nodeInfo.Node()

	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), node, buf, bufLimit))
}

func k8sApiNodeNameFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeName := paramsFromContext(ctx).nodeName

	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), nodeName, buf, bufLimit))
}

func k8sApiPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pod := paramsFromContext(ctx).pod
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), pod, buf, bufLimit))
}

// k8sSchedulerResultClusterEventsFn is a function used by the wasm guest to set the
// cluster events result from guestExportEnqueue.
func k8sSchedulerResultClusterEventsFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	ptr := uint32(stack[0])
	size := uint32(stack[1])

	var clusterEvents []framework.ClusterEvent
	if b, ok := mod.Memory().Read(ptr, size); !ok {
		panic("out of memory reading clusterEvents")
	} else {
		clusterEvents = decodeClusterEvents(b)
	}
	paramsFromContext(ctx).resultClusterEvents = clusterEvents
}

// k8sSchedulerResultNodeNamesFn is a function used by the wasm guest to set the
// node names result from guestExportPreFilter.
func k8sSchedulerResultNodeNamesFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	ptr := uint32(stack[0])
	size := uint32(stack[1])

	var nodeNames []string
	if b, ok := mod.Memory().Read(ptr, size); !ok {
		panic("out of memory reading nodeNames")
	} else {
		nodeNames = fromNULTerminated(b)
	}
	paramsFromContext(ctx).resultNodeNames = nodeNames
}

// k8sSchedulerResultStatusReasonFn is a function used by the wasm guest to set the
// framework.Status reason result from all functions.
func k8sSchedulerResultStatusReasonFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	ptr := uint32(stack[0])
	size := uint32(stack[1])

	var reason string
	if b, ok := mod.Memory().Read(ptr, size); !ok {
		// don't panic if we can't read the message.
		reason = "BUG: out of memory reading message"
	} else {
		reason = string(b)
	}
	paramsFromContext(ctx).resultStatusReason = reason
}
