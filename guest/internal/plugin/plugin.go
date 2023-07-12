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

// Package plugin includes utilities needed for any api.Plugin.
package plugin

import (
	"reflect"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
)

var current api.Plugin

// MustSet sets the plugin once
func MustSet(plugin api.Plugin) {
	if !set(plugin) {
		panic("only one plugin instance is supported")
	}
}

func set(plugin api.Plugin) bool {
	if current == nil {
		current = plugin
		return true
	}
	// current == plugin with the same value works in Go, but not TinyGo.
	return reflect.DeepEqual(current, plugin)
}
