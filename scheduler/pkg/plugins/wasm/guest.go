package wasm

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
)

const (
	guestExportFilter = "filter"
	guestExportMemory = "memory"
)

type guest struct {
	guest    wazeroapi.Module
	filterFn wazeroapi.Function
}

func compileGuest(ctx context.Context, runtime wazero.Runtime, guestBin []byte) (guest wazero.CompiledModule, err error) {
	if guest, err = runtime.CompileModule(ctx, guestBin); err != nil {
		err = fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if handleRequest, ok := guest.ExportedFunctions()[guestExportFilter]; !ok {
		err = fmt.Errorf("wasm: guest doesn't export func[%s]", guestExportFilter)
	} else if len(handleRequest.ParamTypes()) != 0 || !bytes.Equal(handleRequest.ResultTypes(), []wazeroapi.ValueType{i32}) {
		err = fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i32)", guestExportFilter)
	} else if _, ok = guest.ExportedMemories()[guestExportMemory]; !ok {
		err = fmt.Errorf("wasm: guest doesn't export memory[%s]", guestExportMemory)
	}
	return
}

func (pl *wasmPlugin) newGuest(ctx context.Context) (*guest, error) {
	// Concurrent modules can conflict on name. Make sure we have a unique one.
	instanceNum := pl.instanceCounter.Add(1)
	instanceName := pl.guestName + "-" + strconv.FormatUint(instanceNum, 10)
	guestModuleConfig := pl.guestModuleConfig.WithName(instanceName)

	g, err := pl.runtime.InstantiateModule(ctx, pl.guestModule, guestModuleConfig)
	if err != nil {
		_ = pl.runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &guest{guest: g, filterFn: g.ExportedFunction(guestExportFilter)}, nil
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
