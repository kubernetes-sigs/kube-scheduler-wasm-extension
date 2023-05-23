package plugin

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

const (
	i32                      = wazeroapi.ValueTypeI32
	k8sApi                   = "k8s.io/api"
	k8sApiNodeInfoNode       = "nodeInfo/node"
	k8sApiPodSpec            = "pod/spec"
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
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sApiPodSpecFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiPodSpec).
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
	// TODO: fields are different between v1.Node and V1Node
	// Currently mapping node.Name -> V1Node.Metadata.Name
	var msg protoapi.IoK8SApiCoreV1Node
	msg.Metadata = &protoapi.IoK8SApimachineryPkgApisMetaV1ObjectMeta{Name: node.Name}
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), &msg, buf, bufLimit))
}

func k8sApiPodSpecFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pod := filterArgsFromContext(ctx).pod
	// TODO wire types are not in the same order between v1.Pod and V1PodSpec
	var msg protoapi.IoK8SApiCoreV1PodSpec
	msg.NodeName = pod.Spec.NodeName

	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), &msg, buf, bufLimit))
}
