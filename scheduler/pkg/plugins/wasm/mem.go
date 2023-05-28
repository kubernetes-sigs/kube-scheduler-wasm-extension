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

import wazeroapi "github.com/tetratelabs/wazero/api"

// bufLimit is the possibly zero maximum length of a result value to write in
// bytes. If the actual value is larger than this, nothing is written to
// memory.
type bufLimit = uint32

type valueType interface {
	Size() (n int)
	MarshalToSizedBuffer(dAtA []byte) (int, error)
}

func marshalIfUnderLimit(mem wazeroapi.Memory, vt valueType, buf uint32, bufLimit bufLimit) (vLen int) {
	// First, see if the caller passed enough memory to serialize the object.
	vLen = vt.Size()
	if vLen > int(bufLimit) {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}

	// Write directly to the wasm memory.
	wasmMem, ok := mem.Read(buf, uint32(vLen))
	if !ok { // caller passed a length outside memory
		panic("out of memory")
	}
	if _, err := vt.MarshalToSizedBuffer(wasmMem); err != nil {
		panic(err)
	}
	return
}
