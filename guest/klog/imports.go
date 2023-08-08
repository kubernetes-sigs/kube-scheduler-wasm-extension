//go:build tinygo.wasm

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

import "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/internal"

//go:wasmimport k8s.io/klog log
func log(severity internal.Severity, ptr, size uint32)

//go:wasmimport k8s.io/klog logs
func logs(severity internal.Severity, msgPtr, msgSize, kvsPtr, kvsSize uint32)

//go:wasmimport k8s.io/klog severity
func severity() internal.Severity
