module sigs.k8s.io/kube-scheduler-wasm-extension/examples/noop

// Match highest TinyGo's supported version of Go: 1.19 as of TinyGo 0.27
go 1.19

require sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-00010101000000-000000000000

require (
	github.com/magefile/mage v1.14.0 // indirect
	github.com/wasilibs/nottinygc v0.3.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000 // indirect
)

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../../guest

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../../kubernetes/proto
