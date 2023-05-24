module sigs.k8s.io/kube-scheduler-wasm-extension/example

go 1.19

require sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-00010101000000-000000000000

require (
	github.com/knqyf263/go-plugin v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000 // indirect
)

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../guest

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../kubernetes/proto
