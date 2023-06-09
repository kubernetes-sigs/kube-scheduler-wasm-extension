//go:build !tinygo.wasm

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

package imports

// _statusReason is stubbed for compilation outside TinyGo.
func _statusReason(uint32, uint32) {}

// _nodeInfoNode is stubbed for compilation outside TinyGo.
func _nodeInfoNode(uint32, bufLimit) (len uint32) { return }

// _pod is stubbed for compilation outside TinyGo.
func _pod(uint32, bufLimit) (len uint32) { return }
