package plugin

import wazeroapi "github.com/tetratelabs/wazero/api"

// bufLimit is the possibly zero maximum length of a result value to write in
// bytes. If the actual value is larger than this, nothing is written to
// memory.
type bufLimit = uint32

type valueType interface {
	SizeVT() (n int)
	MarshalToSizedBufferVT(dAtA []byte) (int, error)
}

func marshalIfUnderLimit(mem wazeroapi.Memory, vt valueType, buf uint32, bufLimit bufLimit) (vLen int) {
	// First, see if the caller passed enough memory to serialize the object.
	vLen = vt.SizeVT()
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
	if _, err := vt.MarshalToSizedBufferVT(wasmMem); err != nil {
		panic(err)
	}
	return
}
