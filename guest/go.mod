module sigs.k8s.io/kube-scheduler-wasm-extension/guest

// Match highest TinyGo's supported version of Go: 1.19 as of TinyGo 0.27
go 1.19

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ../kubernetes/proto

require sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000

require (
	github.com/google/go-cmp v0.5.9 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
