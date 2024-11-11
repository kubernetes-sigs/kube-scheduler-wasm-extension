module sigs.k8s.io/kube-scheduler-wasm-extension/examples/advanced

go 1.22.0

require (
	github.com/wasilibs/nottinygc v0.4.0
	sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-00010101000000-000000000000
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000
)

require (
	github.com/magefile/mage v1.14.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../../guest

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../../kubernetes/proto
