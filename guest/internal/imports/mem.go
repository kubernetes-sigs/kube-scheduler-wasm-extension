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

import "unsafe"

// BufLimit is the possibly zero maximum length of a result value to write in
// bytes. If the actual value is larger than this, nothing is written to
// memory.
type bufLimit = uint32

var (
	// readBuf is sharable because there is no parallelism in wasm.
	readBuf = make([]byte, readBufLimit)
	// ReadBufPtr is used to avoid duplicate host function calls.
	readBufPtr = uintptr(unsafe.Pointer(&readBuf[0]))
	// ReadBufLimit is constant memory overhead for reading fields.
	readBufLimit = uint32(2048)
)

// stringToPtr returns a pointer and size pair for the given string in a way
// compatible with WebAssembly numeric types.
// The returned pointer aliases the string hence the string must be kept alive
// until ptr is no longer needed.
func stringToPtr(s string) (uint32, uint32) {
	ptr := unsafe.Pointer(unsafe.StringData(s))
	return uint32(uintptr(ptr)), uint32(len(s))
}

// update is for decoding values from memory. The updater doesn't keep a reference to the underlying bytes, so we
// don't need to copy them.
func update(
	fn func(ptr uint32, limit bufLimit) (len uint32),
	updater func([]byte) error) error {

	// Run the update function, which returns the size needed, possibly larger
	// than our buffer.
	size := fn(uint32(readBufPtr), readBufLimit)

	// When the size fits in our buffer, run the update function as the
	// data have been copied to it.
	if size <= readBufLimit {
		return updater(readBuf[:size])
	}

	// If the size in bytes to update is larger than our read buffer, make it
	// larger. This avoids having to garbage collect between larger types.
	readBufLimit = size
	readBuf = make([]byte, readBufLimit)
	readBufPtr = uintptr(unsafe.Pointer(&readBuf[0]))

	// Run the function again to copy data into the correct size buffer.
	_ = fn(uint32(readBufPtr), readBufLimit)
	return updater(readBuf)
}

func getString(fn func(ptr uint32, limit bufLimit) (len uint32)) string {
	size := fn(uint32(readBufPtr), readBufLimit)
	if size == 0 {
		return ""
	}

	// If the function result fit in our read buffer, copy it out.
	if size <= readBufLimit {
		return string(readBuf[:size])
	}

	// If the size returned from the function was larger than our read buffer,
	// we need to execute it again. Make a buffer of exactly the right size.
	buf := make([]byte, size)
	ptr := unsafe.Pointer(&buf[0])
	_ = fn(uint32(uintptr(ptr)), size)
	return unsafe.String((*byte)(ptr), size /* unsafe.IntegerType */)
}
