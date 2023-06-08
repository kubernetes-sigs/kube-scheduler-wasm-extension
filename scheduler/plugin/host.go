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
	i32, i64                 = wazeroapi.ValueTypeI32, wazeroapi.ValueTypeI64
	k8sApi                   = "k8s.io/api"
	k8sApiNodeInfoNode       = "nodeInfo/node"
	k8sApiNodeName           = "nodeName"
	k8sApiPod                = "pod"
	k8sScheduler             = "k8s.io/scheduler"
	k8sSchedulerStatusReason = "status_reason"
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
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerStatusReasonFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerStatusReason).
		Instantiate(ctx)
}

// paramsKey is a context.Context value associated with a params
// pointer to the current request.
type paramsKey struct{}

type params struct {
	// pod is used by guest.filterFn and guest.scoreFn
	pod *v1.Pod

	// nodeInfo is used by guest.filterFn
	nodeInfo *framework.NodeInfo

	// nodeName is used by guest.scoreFn
	nodeName string

	// reason returned by all guest exports.
	//
	// It is a field to avoid compiler-specific malloc/free functions, and to
	// avoid having to deal with out-params because TinyGo only supports a
	// single result.
	reason string
}

func paramsFromContext(ctx context.Context) *params {
	return ctx.Value(paramsKey{}).(*params)
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

// k8sSchedulerStatusReasonFn is a function used by the wasm guest to set the
// framework.Status reason.
func k8sSchedulerStatusReasonFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	ptr := uint32(stack[0])
	size := bufLimit(stack[1])

	var reason string
	if b, ok := mod.Memory().Read(ptr, size); !ok {
		// don't panic if we can't read the message.
		reason = "BUG: out of memory reading message"
	} else {
		reason = string(b)
	}
	paramsFromContext(ctx).reason = reason
}
