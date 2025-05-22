module sigs.k8s.io/kube-scheduler-wasm-extension/examples/nodeports

go 1.22.0

require (
	sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-20250121124236-cb3868918ec5
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-20250121124236-cb3868918ec5
)

require google.golang.org/protobuf v1.30.0 // indirect

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../../kubernetes/proto

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../../guest
