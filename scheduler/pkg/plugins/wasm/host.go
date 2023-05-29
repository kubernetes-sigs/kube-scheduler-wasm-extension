package wasm

import (
	"context"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	i32                      = wazeroapi.ValueTypeI32
	k8sApi                   = "k8s.io/api"
	k8sApiNodeInfoNode       = "nodeInfo/node"
	k8sApiPod                = "pod"
	k8sScheduler             = "k8s.io/scheduler"
	k8sSchedulerStatusReason = "status_reason"
)

func instantiateHostScheduler(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder(k8sScheduler).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerStatusReasonFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("ptr", "size").Export(k8sSchedulerStatusReason).
		Instantiate(ctx)
}

func instantiateHostApi(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder(k8sApi).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiNodeInfoNodeFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiNodeInfoNode).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiPodFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiPod).
		Instantiate(ctx)
}

// filterArgsKey is a context.Context value associated with a filterArgs
// pointer to the current request.
type filterArgsKey struct{}

type filterArgs struct {
	pod      *v1.Pod
	nodeInfo *framework.NodeInfo
	// reason is a field to avoid compiler-specific malloc/free functions, and
	// to avoid having to deal with out-params because TinyGo only supports a
	// single result.
	reason string
}

func filterArgsFromContext(ctx context.Context) *filterArgs {
	return ctx.Value(filterArgsKey{}).(*filterArgs)
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
	filterArgsFromContext(ctx).reason = reason
}

func k8sApiNodeInfoNodeFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	node := filterArgsFromContext(ctx).nodeInfo.Node()

	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), node, buf, bufLimit))
}

func k8sApiPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pod := filterArgsFromContext(ctx).pod
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), pod, buf, bufLimit))
}
