//go:build !wasm

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

// log is stubbed for compilation outside TinyGo.
func log(severity internal.Severity, ptr, size uint32) {}

// logs is stubbed for compilation outside TinyGo.
func logs(severity internal.Severity, msgPtr, msgSize, kvsPtr, kvsSize uint32) {}

// severity is stubbed for compilation outside TinyGo.
func severity() (severity internal.Severity) { return }
