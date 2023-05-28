//go:build !tinygo.wasm

package imports

// reason is stubbed for compilation outside TinyGo.
func _statusReason(ptr, size uint32) {}

// nodeInfoNodeName is stubbed for compilation outside TinyGo.
func _nodeInfoNode(ptr uint32, limit bufLimit) (len uint32) {
	return 0
}

// pod is stubbed for compilation outside TinyGo.
func _pod(ptr uint32, limit bufLimit) (len uint32) {
	return 0
}
