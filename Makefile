%/main.wasm: %/main.go
	@(cd $(@D); tinygo build -o main.wasm -scheduler=none --no-debug -target=wasi main.go)

.PHONY: build-tinygo
build-tinygo: examples/filter-simple/main.wasm examples/noop/main.wasm

%/main-debug.wasm: %/main.go
	@(cd $(@D); tinygo build -o main-debug.wasm -scheduler=none -target=wasi main.go)

.PHONY: profile
profile: examples/filter-simple/main-debug.wasm
	@cd ./internal/e2e; \
	go run ./profiler/profiler.go ../../$^; \
	go tool pprof -text cpu.pprof; \
	go tool pprof -text mem.pprof; \
	rm cpu.pprof mem.pprof

.PHONY: bench-plugin
bench-plugin:
	@(cd internal/e2e; go test -run='^$$' -bench '^BenchmarkPlugin.*$$' . -count=6)

.PHONY: proto-tools
proto-tools:
	cd ./kubernetes/proto/tools; \
	cat tools.go | grep "_" | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cp ./kubernetes/kubernetes.checkout ./.git/modules/kubernetes/kubernetes/info/sparse-checkout
	cd ./kubernetes/kubernetes; \
	git config core.sparsecheckout true; \
	git read-tree -mu HEAD

# This uses the exact generated protos from Kubernetes source, to ensure exact
# wire-type parity. Otherwise, we need expensive to maintain conversion logic.
# We can't use the go generated in the same source tree in TinyGo, because it
# hangs compiling. Instead, we generate UnmarshalVT with go-plugin which is
# known to work with TinyGo.
.PHONY: update-kubernetes-proto
update-kubernetes-proto: proto-tools
	echo "Regenerate the Go protobuf code."
	cd kubernetes/kubernetes/staging/src/; \
	protoc ./k8s.io/apimachinery/pkg/api/resource/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/api/resource/generated.proto=./resource; \
	protoc ./k8s.io/apimachinery/pkg/runtime/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/generated.proto=./runtime; \
	protoc ./k8s.io/apimachinery/pkg/runtime/schema/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/schema/generated.proto=./schema; \
	protoc ./k8s.io/apimachinery/pkg/apis/meta/v1/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/apis/meta/v1/generated.proto=./meta \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/runtime \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/schema/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/schema; \
	protoc ./k8s.io/apimachinery/pkg/util/intstr/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/util/intstr/generated.proto=./instr; \
	protoc ./k8s.io/api/core/v1/generated.proto --go-plugin_out=../../../proto \
		--go-plugin_opt=Mk8s.io/api/core/v1/generated.proto=./api \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/api/resource/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/resource \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/apis/meta/v1/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/runtime \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/runtime/schema/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/schema \
		--go-plugin_opt=Mk8s.io/apimachinery/pkg/util/intstr/generated.proto=sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/instr;
