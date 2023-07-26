package api

// Host is the WebAssembly host side of the scheduler. Specifically, this
// scheduler framework.Plugin written in Go, which runs the Plugin this SDK
// compiles to Wasm.
type Host interface {
	// GetConfig reads any configuration set by the host.
	//
	// Note: This is not updated dynamically.
	GetConfig() []byte
	// ^-- Note: This is a []byte, not a string, for json.Unmarshaler.
}
