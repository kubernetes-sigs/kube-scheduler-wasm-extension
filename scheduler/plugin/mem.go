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

func marshalIfUnderLimit(mem wazeroapi.Memory, vt valueType, buf uint32, bufLimit bufLimit) int {
	// First, see if the caller passed enough memory to serialize the object.
	vLen := vt.Size()
	if vLen == 0 {
		return 0 // nothing to write
	}

	// Next, see if the value will fit inside the buffer.
	if vLen > int(bufLimit) {
		// If it doesn't fit, the caller can decide to retry with a larger
		// buffer or fail.
		return vLen
	}

	// Now, we know the value isn't too large to fit in the buffer. Write it
	// directly to the Wasm memory.
	if wasmMem, ok := mem.Read(buf, uint32(vLen)); !ok {
		panic("out of memory") // Bug: caller passed a length outside memory
	} else if _, err := vt.MarshalToSizedBuffer(wasmMem); err != nil {
		panic(err) // Bug: in marshaller.
	}

	// Success: return the bytes written, so that the caller can unmarshal from
	// a sized buffer.
	return vLen
}
