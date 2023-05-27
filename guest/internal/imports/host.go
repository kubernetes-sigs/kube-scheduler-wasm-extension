package imports

import "runtime"

// StatusReason overwrites the status reason
func StatusReason(reason string) {
	ptr, size := stringToPtr(reason)
	_statusReason(ptr, size)
	runtime.KeepAlive(reason) // keep reason alive until ptr is no longer needed.
}

func NodeInfoNode() []byte {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return getBytes(func(ptr uint32, limit bufLimit) (len uint32) {
		return _nodeInfoNode(ptr, limit)
	})
}

func Pod() []byte {
	// Wrap to avoid TinyGo 0.27: cannot use an exported function as value
	return getBytes(func(ptr uint32, limit bufLimit) (len uint32) {
		return _pod(ptr, limit)
	})
}
