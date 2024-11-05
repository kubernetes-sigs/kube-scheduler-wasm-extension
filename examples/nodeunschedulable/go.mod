module sigs.k8s.io/kube-scheduler-wasm-extension/examples/nodeunschedulable

go 1.20

replace sigs.k8s.io/kube-scheduler-wasm-extension/guest => ./../../guest

replace sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ./../../kubernetes/proto

require (
	github.com/wasilibs/nottinygc v0.7.1
	sigs.k8s.io/kube-scheduler-wasm-extension/guest v0.0.0-00010101000000-000000000000
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/magefile/mage v1.14.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
