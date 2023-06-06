module sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto

// TinyGo 0.27 doesn't fully support Go 1.20, but it supports what we need.
// Particularly, unsafe.SliceData, unsafe.StringData were added to TinyGo 0.27.
// See https://github.com/tinygo-org/tinygo/commit/c43958972c3ffcd51e65414a346e53779edb9f97
go 1.20

require google.golang.org/protobuf v1.30.0
