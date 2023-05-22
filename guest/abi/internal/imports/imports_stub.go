//go:build !tinygo.wasm

package imports

// reason is stubbed for compilation outside TinyGo.
func _reason(ptr, size uint32) {}

// nodeInfoNodeName is stubbed for compilation outside TinyGo.
func _nodeInfoNodeName(ptr uint32, limit bufLimit) (len uint32) {
	return 0
}

// podSpec is stubbed for compilation outside TinyGo.
func _podSpec(ptr uint32, limit bufLimit) (len uint32) {
	return 0
}
