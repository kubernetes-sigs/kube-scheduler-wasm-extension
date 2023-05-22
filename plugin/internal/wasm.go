package internal

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	PluginName = "wasm"
)

func CompileGuest(ctx context.Context, guestPath string) (wazero.Runtime, wazero.CompiledModule, error) {
	guest, err := os.ReadFile(guestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("wasm: error reading guest binary: %w", err)
	}

	// Create the runtime, which when closed releases any resources associated with it.
	runtime := wazero.NewRuntime(ctx)

	// Compile the module, which reduces execution time of Invoke
	module, err := runtime.CompileModule(ctx, guest)
	if err != nil {
		_ = runtime.Close(context.Background())
		return nil, nil, fmt.Errorf("wasm: error compiling binary: %w", err)
	}

	if DetectImports(module.ImportedFunctions())&ModuleWasiP1 != 0 {
		_, err = wasi_snapshot_preview1.Instantiate(ctx, runtime)
	}

	if err != nil {
		_ = runtime.Close(context.Background())
		return nil, nil, fmt.Errorf("wasm: error instantiating host functions: %w", err)
	}

	return runtime, module, nil
}

type ImportModules uint

const (
	ModuleWasiP1 ImportModules = 1 << iota
	ModuleK8sApi
	ModuleK8sScheduler
)

func DetectImports(imports []api.FunctionDefinition) (modules ImportModules) {
	for _, f := range imports {
		moduleName, _, _ := f.Import()
		switch moduleName {
		case "k8s.io/api":
			modules |= ModuleK8sApi
		case "k8s.io/scheduler":
			modules |= ModuleK8sScheduler
		case wasi_snapshot_preview1.ModuleName:
			modules |= ModuleWasiP1
		}
	}
	return
}
