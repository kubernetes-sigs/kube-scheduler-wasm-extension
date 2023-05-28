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

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/framework"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// Filter is a WebAssembly implementation of framework.FilterPlugin.
type Filter interface {
	Filter(FilterArgs) (statusCode framework.Code, statusReason string)
}

// FilterFunc adapts an ordinary function to a Filter.
type FilterFunc func(FilterArgs) (statusCode framework.Code, statusReason string)

// Filter returns f(a).
func (f FilterFunc) Filter(a FilterArgs) (statusCode framework.Code, statusReason string) {
	return f(a)
}

// FilterArgs are the arguments to a Filter.
//
// Note: The arguments are lazy fetched to avoid overhead for properties not in use.
type FilterArgs interface {
	NodeInfo() framework.NodeInfo
	Pod() *protoapi.Pod
}
