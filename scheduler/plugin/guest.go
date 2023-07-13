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
	guestExportMemory    = "memory"
	guestExportEnqueue   = "enqueue"
	guestExportPreFilter = "prefilter"
	guestExportFilter    = "filter"
	guestExportScore     = "score"
)

type guest struct {
	guest       wazeroapi.Module
	out         *bytes.Buffer
	enqueueFn   wazeroapi.Function
	prefilterFn wazeroapi.Function
	filterFn    wazeroapi.Function
	scoreFn     wazeroapi.Function
	callStack   []uint64
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
	// The name isn't important, but it needs to be unique.
	instanceNum := pl.instanceCounter.Add(1)
	moduleConfig := pl.guestModuleConfig.WithName(strconv.FormatUint(instanceNum, 10))

	// A guest may have an instantiation error, which writes to stdout or stderr.
	// Capture stdout and stderr during instantiation.
	var out bytes.Buffer
	moduleConfig = moduleConfig.WithStdout(&out).WithStderr(&out)

	// Set any args used for testing
	moduleConfig = moduleConfig.WithArgs(pl.guestArgs...)

	g, err := pl.runtime.InstantiateModule(ctx, pl.guestModule, moduleConfig)
	if err != nil {
		_ = pl.runtime.Close(ctx)
		return nil, decorateError(&out, "instantiate", err)
	} else {
		out.Reset()
	}

	// Allocate a call stack sized to max of params / return values of any
	// guest function.
	callStack := make([]uint64, 1)

	return &guest{
		guest:       g,
		out:         &out,
		enqueueFn:   g.ExportedFunction(guestExportEnqueue),
		prefilterFn: g.ExportedFunction(guestExportPreFilter),
		filterFn:    g.ExportedFunction(guestExportFilter),
		scoreFn:     g.ExportedFunction(guestExportScore),
		callStack:   callStack,
	}, nil
}

// eventsToRegister calls guestExportEnqueue.
func (g *guest) eventsToRegister(ctx context.Context) []framework.ClusterEvent {
	defer g.out.Reset()
	callStack := g.callStack
	if err := g.enqueueFn.CallWithStack(ctx, callStack); err != nil {
		// framework.EnqueueExtensions.EventsToRegister() does not return an error
		panic(err)
	}
	return paramsFromContext(ctx).resultClusterEvents
}

// preFilter calls guestExportPreFilter.
func (g *guest) preFilter(ctx context.Context) ([]string, *framework.Status) {
	defer g.out.Reset()
	callStack := g.callStack

	if err := g.prefilterFn.CallWithStack(ctx, callStack); err != nil {
		return nil, framework.AsStatus(decorateError(g.out, guestExportPreFilter, err))
	}
	nodeNames := paramsFromContext(ctx).resultNodeNames
	statusCode := int32(callStack[0])
	statusReason := paramsFromContext(ctx).resultStatusReason
	return nodeNames, framework.NewStatus(framework.Code(statusCode), statusReason)
}

// filter calls guestExportFilter.
func (g *guest) filter(ctx context.Context) *framework.Status {
	defer g.out.Reset()
	callStack := g.callStack

	if err := g.filterFn.CallWithStack(ctx, callStack); err != nil {
		return framework.AsStatus(decorateError(g.out, guestExportFilter, err))
	}
	statusCode := int32(callStack[0])
	statusReason := paramsFromContext(ctx).resultStatusReason
	return framework.NewStatus(framework.Code(statusCode), statusReason)
}

// score calls guestExportScore.
func (g *guest) score(ctx context.Context) (int64, *framework.Status) {
	defer g.out.Reset()
	callStack := g.callStack

	if err := g.scoreFn.CallWithStack(ctx, callStack); err != nil {
		return 0, framework.AsStatus(decorateError(g.out, guestExportScore, err))
	}

	score := int32(callStack[0] >> 32)
	statusCode := int32(callStack[0])
	statusReason := paramsFromContext(ctx).resultStatusReason
	return int64(score), framework.NewStatus(framework.Code(statusCode), statusReason)
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

type interfaces uint

const (
	iEnqueueExtensions interfaces = 1 << iota
	iPreFilterPlugin
	iFilterPlugin
	iScorePlugin
)

func detectInterfaces(exportedFns map[string]wazeroapi.FunctionDefinition) (interfaces, error) {
	var e interfaces
	for name, f := range exportedFns {
		switch name {
		case guestExportEnqueue:
			if len(f.ParamTypes()) != 0 || len(f.ResultTypes()) != 0 {
				return 0, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> ()", name)
			}
			e |= iEnqueueExtensions
		case guestExportPreFilter:
			if len(f.ParamTypes()) != 0 || !bytes.Equal(f.ResultTypes(), []wazeroapi.ValueType{i32}) {
				return 0, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i32)", name)
			}
			e |= iPreFilterPlugin
		case guestExportFilter:
			if len(f.ParamTypes()) != 0 || !bytes.Equal(f.ResultTypes(), []wazeroapi.ValueType{i32}) {
				return 0, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i32)", name)
			}
			e |= iFilterPlugin
		case guestExportScore:
			if len(f.ParamTypes()) != 0 || !bytes.Equal(f.ResultTypes(), []wazeroapi.ValueType{i64}) {
				return 0, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i64)", name)
			}
			e |= iScorePlugin
		}
	}
	if e == 0 {
		return 0, fmt.Errorf("wasm: guest does not export any plugin functions")
	}
	return e, nil
}
