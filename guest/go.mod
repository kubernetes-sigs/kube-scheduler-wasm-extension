module sigs.k8s.io/kube-scheduler-wasm-extension/guest

go 1.19

replace (
	google.golang.org/grpc => google.golang.org/grpc v1.38.0
	k8s.io/api => k8s.io/api v0.26.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.26.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.26.2
	k8s.io/apiserver => k8s.io/apiserver v0.26.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.26.2
	k8s.io/client-go => k8s.io/client-go v0.26.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.26.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.26.2
	k8s.io/code-generator => k8s.io/code-generator v0.26.2
	k8s.io/component-base => k8s.io/component-base v0.26.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.26.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.26.2
	k8s.io/cri-api => k8s.io/cri-api v0.26.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.26.2
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.26.2
	k8s.io/kms => k8s.io/kms v0.26.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.26.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.26.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.26.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.26.2
	k8s.io/kubectl => k8s.io/kubectl v0.26.2
	k8s.io/kubelet => k8s.io/kubelet v0.26.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.26.2
	k8s.io/metrics => k8s.io/metrics v0.26.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.26.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.26.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.26.2
	sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto => ../kubernetes/proto
)

require sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto v0.0.0-00010101000000-000000000000

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/knqyf263/go-plugin v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
