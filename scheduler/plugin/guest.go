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
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	guestExportFilter = "filter"
	guestExportMemory = "memory"
)

type guest struct {
	guest    wazeroapi.Module
	out      *bytes.Buffer
	filterFn wazeroapi.Function
}

func compileGuest(ctx context.Context, runtime wazero.Runtime, guestBin []byte) (guest wazero.CompiledModule, err error) {
	if guest, err = runtime.CompileModule(ctx, guestBin); err != nil {
		err = fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if _, ok := guest.ExportedMemories()[guestExportMemory]; !ok {
		err = fmt.Errorf("wasm: guest doesn't export memory[%s]", guestExportMemory)
	}
	return
}

func (pl *wasmPlugin) newGuest(ctx context.Context) (*guest, error) {
	// Concurrent modules can conflict on name. Make sure we have a unique one.
	instanceNum := pl.instanceCounter.Add(1)
	instanceName := pl.guestName + "-" + strconv.FormatUint(instanceNum, 10)
	guestModuleConfig := pl.guestModuleConfig.WithName(instanceName)

	// A guest may have an instantiation error, which writes to stdout or stderr.
	// Capture stdout and stderr during instantiation.
	var out bytes.Buffer
	guestModuleConfig = guestModuleConfig.WithStdout(&out).WithStderr(&out)

	g, err := pl.runtime.InstantiateModule(ctx, pl.guestModule, guestModuleConfig)
	if err != nil {
		_ = pl.runtime.Close(ctx)
		return nil, decorateError(&out, "instantiate", err)
	} else {
		out.Reset()
	}

	return &guest{
		guest:    g,
		out:      &out,
		filterFn: g.ExportedFunction(guestExportFilter),
	}, nil
}

// filter calls the WebAssembly guest function handler.FuncHandleRequest.
func (g *guest) filter(ctx context.Context) *framework.Status {
	defer g.out.Reset()
	if results, err := g.filterFn.Call(ctx); err != nil {
		return framework.AsStatus(decorateError(g.out, "filter", err))
	} else {
		code := uint32(results[0])
		reason := filterParamsFromContext(ctx).reason
		return framework.NewStatus(framework.Code(code), reason)
	}
}

func decorateError(out fmt.Stringer, fn string, err error) error {
	detail := out.String()
	if detail != "" {
		err = fmt.Errorf("wasm: %s error: %s\n%v", fn, detail, err)
	} else {
		err = fmt.Errorf("wasm: %s error: %v", fn, err)
	}
	return err
}

type exports uint

const (
	exportFilterPlugin exports = 1 << iota
)

func detectExports(exportedFns map[string]wazeroapi.FunctionDefinition) (exports, error) {
	var e exports
	for name, f := range exportedFns {
		switch name {
		case guestExportFilter:
			if len(f.ParamTypes()) != 0 || !bytes.Equal(f.ResultTypes(), []wazeroapi.ValueType{i32}) {
				return 0, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i32)", guestExportFilter)
			}
			e |= exportFilterPlugin
		}
	}
	if e == 0 {
		return 0, fmt.Errorf("wasm: guest does not export any plugin functions")
	}
	return e, nil
}
