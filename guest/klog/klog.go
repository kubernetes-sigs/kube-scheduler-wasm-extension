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

package klog

import (
	"fmt"
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/internal"
)

// Get can be called at any time, to get a logger that writes to the
// WebAssembly host.
//
// For example:
//
//	func main() {
//		klog := klog.Get()
//		klog.Info("hello", "world")
//	}
func Get() api.Klog {
	return instance
}

var instance api.Klog = &internal.Klog{
	Severity: severity(),
	LogFn:    logFn,
	LogSFn:   logSFn,
}

func logFn(severity internal.Severity, msg []byte) {
	ptr, size := mem.BytesToPtr(msg)
	log(severity, ptr, size)
	runtime.KeepAlive(msg) // keep msg alive until ptr is no longer needed.
}

func logSFn(severity internal.Severity, msg string, kvs []byte) {
	msgPtr, msgSize := mem.StringToPtr(msg)
	kvsPtr, kvsSize := mem.BytesToPtr(kvs)
	logs(severity, msgPtr, msgSize, kvsPtr, kvsSize)
	runtime.KeepAlive(msg) // keep msg alive until ptr is no longer needed.
	runtime.KeepAlive(kvs) // keep kvs alive until ptr is no longer needed.
}

// KObj is a convenience that calls api.KObj. This is re-declared here for
// familiarity.
//
// Note: See Info for unit test and benchmarking impact.
func KObj(obj proto.Metadata) fmt.Stringer {
	return api.KObj(obj)
}

// KObjSlice is a convenience that calls api.KObjSlice. This is re-declared
// here for familiarity.
//
// Note: See Info for unit test and benchmarking impact.
func KObjSlice[M proto.Metadata](objs []M) fmt.Stringer {
	return api.KObjSlice(objs)
}

// KObjSliceFn is a convenience that calls api.KObjSliceFn. This is re-declared here
// for familiarity.
//
// Note: See Info for unit test and benchmarking impact.
func KObjSliceFn[M proto.Metadata](lazy func() []M) fmt.Stringer {
	return api.KObjSliceFn(lazy)
}

// Info is a convenience that calls the same method documented on api.Klog.
//
// Note: Code that uses can be unit tested in normal Go, but cannot be unit
// tested or benchmarked via `tinygo test -target=wasi`. To avoid this problem,
// use Get instead.
func Info(args ...any) {
	instance.Info(args...)
}

// InfoS is a convenience that calls the same method documented on api.Klog.
//
// Note: See Info for unit test and benchmarking impact.
func InfoS(msg string, keysAndValues ...any) {
	instance.InfoS(msg, keysAndValues...)
}

// Error is a convenience that calls the same method documented on api.Klog.
//
// Note: See Info for unit test and benchmarking impact.
func Error(args ...any) {
	instance.Error(args...)
}

// ErrorS is a convenience that calls the same method documented on api.Klog.
//
// Note: See Info for unit test and benchmarking impact.
func ErrorS(err error, msg string, keysAndValues ...any) {
	instance.ErrorS(err, msg, keysAndValues...)
}
