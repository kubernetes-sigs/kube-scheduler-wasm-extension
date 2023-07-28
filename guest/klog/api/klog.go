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

package api

// Klog is a logger that exposes functions typically in the klog package.
//
// This contains functions like klog.Info, but allows flexibility to disable
// logging in WebAssembly, where it is more expensive due to inlined garbage
// collection.
//
// Note: Embed UnimplementedKlog when implementing for real or test usage.
type Klog interface {
	// Info records an event at INFO level.
	//
	// The event is a concatenation of the arguments. A newline is appended when
	// the last arg doesn't already end with one.
	//
	// # Notes
	//
	//   - See Klog godoc for an example.
	//   - Wrap args in objectRef where possible, to normalize the output of types
	//     such as proto.Pod.
	Info(args ...any)

	// InfoS is like klog.InfoS. This records the description of an event,
	// `msg` followed by key/value pairs to describe it.
	//
	// # Notes
	//
	//   - See Klog godoc for an example.
	//   - Wrap values in objectRef where possible, to normalize the output of types
	//     such as proto.Pod.
	InfoS(msg string, keysAndValues ...any)

	// Error is like Info, except ERROR level.
	Error(args ...any)

	// ErrorS is like InfoS, except ERROR level. Also, the `err` parameter
	// becomes the value of the "err" key in `keysAndValues`.
	ErrorS(err error, msg string, keysAndValues ...any)
}

type UnimplementedKlog struct{}

// Info implements Klog.Info
func (UnimplementedKlog) Info(args ...any) {}

// InfoS implements Klog.InfoS
func (UnimplementedKlog) InfoS(msg string, keysAndValues ...any) {}

// Error implements Klog.Error
func (UnimplementedKlog) Error(args ...any) {}

// ErrorS implements Klog.ErrorS
func (UnimplementedKlog) ErrorS(err error, msg string, keysAndValues ...any) {}
