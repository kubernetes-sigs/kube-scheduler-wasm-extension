package wasm

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// prepareRuntime compiles the guest and instantiates any host modules it needs.
func prepareRuntime(ctx context.Context, guestPath string) (runtime wazero.Runtime, guest wazero.CompiledModule, err error) {
	guestBin, err := os.ReadFile(guestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("wasm: error reading guest binary: %w", err)
	}

	// Create the runtime, which when closed releases any resources associated with it.
	runtime = wazero.NewRuntime(ctx)

	// Close the runtime on any error
	defer func() {
		if err != nil {
			_ = runtime.Close(context.Background())
			runtime = nil
			return
		}
	}()

	// Compile the guest to ensure any errors are known up front.
	if guest, err = compileGuest(ctx, runtime, guestBin); err != nil {
		return
	}

	// Detect and handle any host imports or lack thereof.
	imports := detectImports(guest.ImportedFunctions())
	switch {
	case imports&importWasiP1 != 0:
		if _, err = wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
			err = fmt.Errorf("wasm: error instantiating wasi: %w", err)
			return
		}
		fallthrough // proceed to more imports
	case imports&importK8sApi != 0:
		if _, err = instantiateHostApi(ctx, runtime); err != nil {
			err = fmt.Errorf("wasm: error instantiating api host functions: %w", err)
			return
		}
		fallthrough // proceed to more imports
	case imports&importK8sScheduler != 0:
		if _, err = instantiateHostScheduler(ctx, runtime); err != nil {
			err = fmt.Errorf("wasm: error instantiating scheduler host functions: %w", err)
			return
		}
	}
	return
}

type imports uint

const (
	importWasiP1 imports = 1 << iota
	importK8sApi
	importK8sScheduler
)

func detectImports(importedFns []api.FunctionDefinition) (imports imports) {
	for _, f := range importedFns {
		moduleName, _, _ := f.Import()
		switch moduleName {
		case k8sApi:
			imports |= importK8sApi
		case k8sScheduler:
			imports |= importK8sScheduler
		case wasi_snapshot_preview1.ModuleName:
			imports |= importWasiP1
		}
	}
	return
}
