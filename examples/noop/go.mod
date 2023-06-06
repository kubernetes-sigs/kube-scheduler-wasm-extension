module sigs.k8s.io/kube-scheduler-wasm-extension/examples/noop

// TinyGo 0.27 doesn't fully support Go 1.20, but it supports what we need.
// Particularly, unsafe.SliceData, unsafe.StringData were added to TinyGo 0.27.
// See https://github.com/tinygo-org/tinygo/commit/c43958972c3ffcd51e65414a346e53779edb9f97
go 1.20

require sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-00010101000000-000000000000

require (
	github.com/magefile/mage v1.14.0 // indirect
	github.com/wasilibs/nottinygc v0.3.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000 // indirect
)

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../../guest

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../../kubernetes/proto
