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

// Package host imports an api.Host from the WebAssembly host.
package host

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
)

// Get can be called at any time, to use features exported by the host.
//
// For example:
//
//	func main() {
//		config := host.Get().GetConfig()
//		// decode yaml
//	}
func Get() api.Host {
	return currentHost
}

var currentHost api.Host = &host{}

type host struct {
	config []byte
}

func (n *host) GetConfig() []byte {
	return n.lazyConfig()
}

func (n *host) lazyConfig() []byte {
	if config := n.config; config != nil {
		return config
	}

	// Wrap to avoid TinyGo 0.28: cannot use an exported function as value
	n.config = mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return getConfig(ptr, limit)
	})
	return n.config
}
