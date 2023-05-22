package vfs

import (
	"bytes"
	"context"
	"io/fs"
	"strconv"
	"sync/atomic"
	"testing/fstest"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/sys"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/plugin/internal"
)

func IsVFSPlugin(guestModule wazero.CompiledModule) bool {
	return internal.DetectImports(guestModule.ImportedFunctions()) == internal.ModuleWasiP1
}

func NewPlugin(_ context.Context, runtime wazero.Runtime, guestModule wazero.CompiledModule, guestName string) (pl framework.Plugin, err error) {
	return &vfsPlugin{
		runtime:           runtime,
		guestName:         guestName,
		guestModule:       guestModule,
		guestModuleConfig: wazero.NewModuleConfig(),
		instanceCounter:   atomic.Uint64{},
	}, nil
}

type vfsPlugin struct {
	runtime           wazero.Runtime
	guestName         string
	guestModule       wazero.CompiledModule
	guestModuleConfig wazero.ModuleConfig
	instanceCounter   atomic.Uint64
}

var _ framework.FilterPlugin = (*vfsPlugin)(nil)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *vfsPlugin) Name() string {
	return internal.PluginName
}

var _ fs.FS = (*vfs)(nil)

// vfs is a virtual file system which allows use of all pod and nodeInfo data
// without custom ABIs. The caller can deserialize the top-level proto, or
// more specific ones.
type vfs struct {
	pod      *v1.Pod
	nodeInfo *framework.NodeInfo
}

func (v vfs) Open(name string) (fs.File, error) {
	var marshaller func() ([]byte, error)
	switch name {
	case "pod/spec":
		// TODO v.pod.Spec.Marshal is incompatible, find a way to automatically
		// convert *v1.PodSpec to protoapi.IoK8SApiCoreV1PodSpec
		var msg protoapi.IoK8SApiCoreV1PodSpec
		msg.NodeName = v.pod.Spec.NodeName
		marshaller = msg.MarshalVT
	case "nodeInfo/node/name":
		marshaller = func() ([]byte, error) {
			return []byte(v.nodeInfo.Node().Name), nil
		}
	default:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if b, err := marshaller(); err != nil {
		return nil, err
	} else {
		return (fstest.MapFS{name: {Data: b}}).Open(name)
	}
}

// Filter invoked at the filter extension point.
func (pl *vfsPlugin) Filter(ctx context.Context, _ *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// Concurrent modules can conflict on name. Make sure we have a unique one.
	instanceNum := pl.instanceCounter.Add(1)
	instanceName := pl.guestName + "-" + strconv.FormatUint(instanceNum, 10)
	guestModuleConfig := pl.guestModuleConfig.WithName(instanceName)

	// Lazy marshal node and pod on-demand
	guestModuleConfig = guestModuleConfig.WithFSConfig(
		wazero.NewFSConfig().
			WithFSMount(&vfs{pod: pod, nodeInfo: nodeInfo}, "/kdev"))

	// Any STDERR will be the status reason
	var stderr bytes.Buffer
	guestModuleConfig = guestModuleConfig.WithStderr(&stderr)

	// Allow the program to inspect the args
	argsSlice := []string{"scheduler", "filter"}
	guestModuleConfig = guestModuleConfig.WithArgs(argsSlice...)

	// Instantiating executes the guest's main function (exported as _start).
	mod, err := pl.runtime.InstantiateModule(ctx, pl.guestModule, guestModuleConfig)
	if err == nil {
		// WASI typically calls proc_exit which exits the guestModule, but just in case
		// it doesn't, close the guestModule manually.
		_ = mod.Close(ctx)
		return nil // success
	}

	if exitErr, ok := err.(*sys.ExitError); ok {
		return framework.NewStatus(framework.Code(exitErr.ExitCode()), stderr.String())
	} else {
		return framework.AsStatus(err)
	}
}

// Close implements io.Closer
func (pl *vfsPlugin) Close() error {
	// wazero's runtime closes everything.
	if rt := pl.runtime; rt != nil {
		return rt.Close(context.Background())
	}
	return nil
}
