package abi

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/internal"
)

// bufLimit is the possibly zero maximum length of a result value to write in
// bytes. If the actual value is larger than this, nothing is written to
// memory.
type bufLimit = uint32

func IsABIPlugin(module wazero.CompiledModule) bool {
	return internal.DetectImports(module.ImportedFunctions())&internal.ModuleK8sScheduler != 0
}

func NewPlugin(ctx context.Context, runtime wazero.Runtime, guestModule wazero.CompiledModule, guestName string) (framework.Plugin, error) {
	if _, err := instantiateHostApi(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating api host functions: %w", err)
	}

	if _, err := instantiateHostScheduler(ctx, runtime); err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating scheduler host functions: %w", err)
	}

	pl := &abiPlugin{
		guestModuleConfig: wazero.NewModuleConfig(),
		guestName:         guestName,
		runtime:           runtime,
		guestModule:       guestModule,
		instanceCounter:   atomic.Uint64{},
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	g, err := pl.getOrCreateGuest(ctx)
	if err != nil {
		return nil, err
	}
	pl.pool.Put(g)

	return pl, nil
}

type guest struct {
	guest    wazeroapi.Module
	filterFn wazeroapi.Function
}

func (pl *abiPlugin) newGuest(ctx context.Context) (*guest, error) {
	// Concurrent modules can conflict on name. Make sure we have a unique one.
	instanceNum := pl.instanceCounter.Add(1)
	instanceName := pl.guestName + "-" + strconv.FormatUint(instanceNum, 10)
	guestModuleConfig := pl.guestModuleConfig.WithName(instanceName)

	g, err := pl.runtime.InstantiateModule(ctx, pl.guestModule, guestModuleConfig)
	if err != nil {
		_ = pl.runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &guest{guest: g, filterFn: g.ExportedFunction("filter")}, nil
}

// filter calls the WebAssembly guest function handler.FuncHandleRequest.
func (g *guest) filter(ctx context.Context) *framework.Status {
	if results, err := g.filterFn.Call(ctx); err != nil {
		return framework.AsStatus(err)
	} else {
		code := uint32(results[0])
		reason := filterArgsFromContext(ctx).reason
		return framework.NewStatus(framework.Code(code), reason)
	}
}

type abiPlugin struct {
	runtime           wazero.Runtime
	guestName         string
	guestModule       wazero.CompiledModule
	guestModuleConfig wazero.ModuleConfig
	instanceCounter   atomic.Uint64
	pool              sync.Pool
}

var _ framework.FilterPlugin = (*abiPlugin)(nil)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *abiPlugin) Name() string {
	return internal.PluginName
}

func (pl *abiPlugin) getOrCreateGuest(ctx context.Context) (*guest, error) {
	poolG := pl.pool.Get()
	if poolG == nil {
		if g, createErr := pl.newGuest(ctx); createErr != nil {
			return nil, createErr
		} else {
			poolG = g
		}
	}
	return poolG.(*guest), nil
}

// Filter invoked at the filter extension point.
func (pl *abiPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	g, err := pl.getOrCreateGuest(ctx)
	if err != nil {
		return framework.AsStatus(err)
	}
	defer pl.pool.Put(g)

	// The guest Wasm may call host functions, so we add context parameters of
	// the current args.
	args := &filterArgs{pod: pod, nodeInfo: nodeInfo}
	ctx = context.WithValue(ctx, filterArgsKey{}, args)
	return g.filter(ctx)
}

// Close implements io.Closer
func (pl *abiPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
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

func schedulerReason(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
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

func apiNodeInfoNodeName(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeInfoNodeName := filterArgsFromContext(ctx).nodeInfo.Node().Name
	nodeInfoNodeNameLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, nodeInfoNodeName)

	stack[0] = uint64(nodeInfoNodeNameLen)
}

func apiPodSpec(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pod := filterArgsFromContext(ctx).pod
	// TODO v.pod.Spec.Marshal is incompatible, find a way to automatically
	// convert *v1.PodSpec to protoapi.IoK8SApiCoreV1PodSpec
	var msg protoapi.IoK8SApiCoreV1PodSpec
	msg.NodeName = pod.Spec.NodeName
	bytes := mustMarshalVT(msg.MarshalVT)
	methodLen := writeBytesIfUnderLimit(mod.Memory(), buf, bufLimit, bytes)

	stack[0] = uint64(methodLen)
}

func mustMarshalVT(marshalVT func() ([]byte, error)) []byte {
	b, err := marshalVT()
	if err != nil {
		panic(err)
	}
	return b
}

func writeBytesIfUnderLimit(mem wazeroapi.Memory, offset uint32, limit bufLimit, v []byte) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mem.Write(offset, v)
	return
}

func writeStringIfUnderLimit(mem wazeroapi.Memory, offset uint32, limit bufLimit, v string) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mem.WriteString(offset, v)
	return
}

const i32 = wazeroapi.ValueTypeI32

func instantiateHostScheduler(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder("k8s.io/scheduler").
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(schedulerReason), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("ptr", "size").Export("reason").
		Instantiate(ctx)
}

func instantiateHostApi(ctx context.Context, runtime wazero.Runtime) (wazeroapi.Module, error) {
	return runtime.NewHostModuleBuilder("k8s.io/api").
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(apiNodeInfoNodeName), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export("nodeInfo/node/name").
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(apiPodSpec), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export("pod/spec").
		Instantiate(ctx)
}
