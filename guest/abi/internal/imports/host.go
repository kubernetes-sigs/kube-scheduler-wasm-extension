package imports

import "runtime"

// Reason overwrites the status reason
func Reason(reason string) {
	ptr, size := stringToPtr(reason)
	_reason(ptr, size)
	runtime.KeepAlive(reason) // keep reason alive until ptr is no longer needed.
}

func NodeInfoNodeName() string {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return getString(func(ptr uint32, limit bufLimit) (len uint32) {
		return _nodeInfoNodeName(ptr, limit)
	})
}

func PodSpec() []byte {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return getBytes(func(ptr uint32, limit bufLimit) (len uint32) {
		return _podSpec(ptr, limit)
	})
}
