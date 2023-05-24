module sigs.k8s.io/kube-scheduler-wasm-extension/guest

go 1.20

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ../kubernetes/proto

require sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/knqyf263/go-plugin v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
