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

type PluginConfig struct {
	// GuestURL is the URL to the guest wasm.
	// Valid schemes are file:// for a local file or http[s]:// for one
	// retrieved via HTTP.
	GuestURL string `json:"guestURL"`

	// GuestConfig is any configuration to give to the guest.
	GuestConfig string `json:"guestConfig"`

	// LogSeverity has the following values:
	//
	//   - 0: info (default)
	//   - 1: warning
	//   - 2: error
	//   - 3: fatal
	LogSeverity int32 `json:"logSeverity"`

	// Args are the os.Args the guest will receive, exposed for tests.
	Args []string
}
