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
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// prepareRuntime compiles the guest and instantiates any host modules it needs.
func prepareRuntime(ctx context.Context, guestBin []byte) (runtime wazero.Runtime, guest wazero.CompiledModule, err error) {
	// Create the runtime, which when closed releases any resources associated with it.
	runtime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		// Here are settings required by the wasm profiler wzprof:
		// * DebugInfo is already true by default, so no impact.
		// * CustomSections buffers more data into memory at compile time.
		WithDebugInfoEnabled(true).WithCustomSections(true))

	// Close the runtime on any error
	defer func() {
		if err != nil {
			_ = runtime.Close(context.Background())
			runtime = nil
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

func detectImports(importedFns []api.FunctionDefinition) imports {
	var imports imports
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
	return imports
}
