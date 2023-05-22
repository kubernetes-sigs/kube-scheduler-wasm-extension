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

func getString(fn func(ptr uint32, limit bufLimit) (len uint32)) (result string) {
	size := fn(uint32(readBufPtr), readBufLimit)
	if size == 0 {
		return
	}
	if size > 0 && size <= readBufLimit {
		return string(readBuf[:size]) // string will copy the buffer.
	}

	// Otherwise, allocate a new string
	buf := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&buf[0]))
	_ = fn(uint32(ptr), size)
	s := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	return *(*string)(unsafe.Pointer(&s))
}

func getBytes(fn func(ptr uint32, limit bufLimit) (len uint32)) (result []byte) {
	size := fn(uint32(readBufPtr), readBufLimit)
	if size == 0 {
		return
	}
	if size > 0 && size <= readBufLimit {
		// copy to avoid passing a mutable buffer
		result = make([]byte, size)
		copy(result, readBuf)
		return
	}
	buf := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&buf[0]))
	_ = fn(uint32(ptr), size)
	return buf
}
