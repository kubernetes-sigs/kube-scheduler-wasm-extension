//go:build tinygo.wasm

package permit

//go:wasmimport k8s.io/scheduler result.timeout
func setTimeoutResult(uint32)
