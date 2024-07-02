gofumpt       := mvdan.cc/gofumpt@v0.5.0
gosimports    := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.2

examples/advanced/main.wasm: examples/advanced/main.go
	@(cd $(@D); tinygo build -o main.wasm -gc=custom -tags=custommalloc -scheduler=none --no-debug -target=wasi .)

%/main.wasm: %/main.go
	@(cd $(@D); tinygo build -o main.wasm -scheduler=none --no-debug -target=wasi .)

.PHONY: build-tinygo
build-tinygo: examples/nodenumber/main.wasm examples/advanced/main.wasm guest/testdata/cyclestate/main.wasm guest/testdata/filter/main.wasm guest/testdata/score/main.wasm \
			  guest/testdata/bind/main.wasm guest/testdata/reserve/main.wasm guest/testdata/handle/main.wasm

%/main-debug.wasm: %/main.go
	@(cd $(@D); tinygo build -o main-debug.wasm -gc=custom -tags=custommalloc -scheduler=none -target=wasi .)

# Testing the guest code means running it with TinyGo, which internally
# compiles the unit tests to a wasm binary, then runs it with wazero.
.PHONY: test-guest
test-guest: guest/.tinygo-target.json
	@(cd guest; tinygo test -v -target .tinygo-target.json ./...)
	@(cd examples/advanced; tinygo test -v -target ../../guest/.tinygo-target.json ./plugin/...)

# Benchmarking the guest code means running it with TinyGo, which internally
# compiles the benchmarks to a wasm binary, then runs it with wazero.
.PHONY: bench-guest
bench-guest: guest/.tinygo-target.json
	@(cd internal/e2e/guest; tinygo test -gc=custom -tags=custommalloc -scheduler=none -v -count=6 -target ../../../guest/.tinygo-target.json -run='^$$' -bench '^Benchmark.*$$' .)

# By default, TinyGo's wasi target uses wasmtime. but our plugin uses wazero.
# This makes a wasi target that uses the same wazero version as the scheduler.
wazero_version := $(shell (cd scheduler; go list -f '{{ .Module.Version }}' github.com/tetratelabs/wazero))
guest/.tinygo-target.json: scheduler/go.mod
	@sed 's~"wasmtime.*"~"go run github.com/tetratelabs/wazero/cmd/wazero@$(wazero_version) run {}"~' $(shell tinygo env TINYGOROOT)/targets/wasi.json > $@

.PHONY: build-wat
build-wat: $(wildcard scheduler/test/testdata/*/*.wat)
	@set -e; for f in $^; do \
        wasm=$$(echo $$f | sed -e 's/\.wat/\.wasm/'); \
		wat2wasm -o $$wasm --debug-names $$f; \
	done

.PHONY: testdata
testdata: build-tinygo build-wat

.PHONY: profile
profile: examples/advanced/main-debug.wasm
	@cd ./internal/e2e; \
	go run ./profiler/profiler.go ../../$^; \
	go tool pprof -text cpu.pprof; \
	go tool pprof -text mem.pprof; \
	rm cpu.pprof mem.pprof

.PHONY: bench-example
bench-example:
	@(cd internal/e2e/scheduler; go test -run='^$$' -bench '^BenchmarkExample.*$$' . -count=6)

.PHONY: scheduler-perf
scheduler-perf:
	@(cd internal/e2e/scheduler_perf; go test -run='^$$' -bench '^BenchmarkPerfScheduling$$' .)

.PHONY: proto-tools
proto-tools:
	cd ./kubernetes/proto/tools; \
	cat tools.go | grep "_" | awk -F'"' '{print $$2}' | xargs -tI % go install %

# Generate protobuf sources from the same kubernetes version as the plugin.
kubernetes_version := v1.27.3
.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cp ./kubernetes/kubernetes.checkout ./.git/modules/kubernetes/kubernetes/info/sparse-checkout
	cd ./kubernetes/kubernetes; \
	git checkout $(kubernetes_version); \
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
	@$(MAKE) format

.PHONY: lint
lint:
	@for f in $(all_mods); do \
        (cd $$(dirname $$f); go run $(golangci_lint) run --timeout 5m); \
        if [ $$? -ne 0 ]; then \
            exit 1; \
        fi; \
	done

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -local sigs.k8s.io/kube-scheduler-wasm-extension/ -w $(shell find . -name '*.go' -type f)

# all_mods are the go modules including examples. Examples should also be
# formatted, lint checked, etc. even if they are are built with TinyGo.
all_mods             := ./internal/e2e/go.mod ./internal/e2e/guest/go.mod ./scheduler/go.mod ./examples/advanced/go.mod ./guest/testdata/go.mod ./guest/go.mod ./kubernetes/proto/go.mod
# all_mods are modules that can't be built with normal Go, such as due to being
# a tool, or a TinyGo main package.
all_unbuildable_mods := ./examples/go.mod ./kubernetes/proto/tools/go.mod

.PHONY: tidy
tidy:
	@for f in $(all_mods); do \
        (cd $$(dirname $$f); go mod tidy); \
        if [ $$? -ne 0 ]; then \
            exit 1; \
        fi; \
	done

.PHONY: build
build:
	@for f in $(filter-out $(all_unbuildable_mods), $(all_mods)); do \
        (cd $$(dirname $$f); go build ./...); \
        if [ $$? -ne 0 ]; then \
            exit 1; \
        fi; \
	done

# Test runs both host and guest unit tests with normal go.
.PHONY: test
test:
	@(cd scheduler; go test -v ./...)
	@(cd guest; go test -v ./...)
	@(cd examples; go test -v ./...)
	@(cd examples/advanced; go test -v ./plugin/...)
	@(cd internal/e2e; go test -v ./...)

.PHONY: check  # Pre-flight check for pull requests
check:
	@# To make troubleshooting easier, order targets from simple to specific.
	@$(MAKE) tidy
	@$(MAKE) build
	@$(MAKE) format
	@$(MAKE) lint
	@if [ ! -z "`git status -s`" ]; then \
		echo "The following differences will fail CI until committed:"; \
		git diff --exit-code; \
	fi
