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

// Package internal allows unit testing without requiring wasm imports.
package internal

import "unsafe"

var (
	// config is lazy read on Get.
	config string
	// configRead is required to differentiate an empty read from never read.
	configRead bool
)

// Get lazy reads configuration from the given function and returns a byte
// slice of the result.
func Get(readConfig func() string) []byte {
	if !configRead {
		config = readConfig()
		configRead = true
	}
	if config == "" {
		return nil // don't call unsafe.StringData("")
	}
	// Return the bytes under `config`. This is safe because `config` is
	// package-scoped, so is always kept alive. This is an alternative to
	// maintaining `mem.GetBytes` or `[]byte(stringConfig)`, which allocates.
	return unsafe.Slice(unsafe.StringData(config), len(config))
}
