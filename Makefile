gofumpt       := mvdan.cc/gofumpt@v0.5.0
gosimports    := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.2

%/main.wasm: %/main.go
	@(cd $(@D); tinygo build -o main.wasm -gc=custom -tags=custommalloc -scheduler=none --no-debug -target=wasi main.go)

.PHONY: build-tinygo
build-tinygo: examples/filter-simple/main.wasm examples/score-simple/main.wasm guest/testdata/noop/main.wasm

%/main-debug.wasm: %/main.go
	@(cd $(@D); tinygo build -o main-debug.wasm -gc=custom -tags=custommalloc -scheduler=none -target=wasi main.go)

.PHONY: build-wat
build-wat: $(wildcard scheduler/test/testdata/*/*.wat)
	@for f in $^; do \
        wasm=$$(echo $$f | sed -e 's/\.wat/\.wasm/'); \
	    wat2wasm -o $$wasm --debug-names $$f; \
	done

.PHONY: testdata
testdata:
	@$(MAKE) build-tinygo
	@$(MAKE) build-wat

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
	@$(MAKE) format

.PHONY: lint
lint:
	@for f in $(all_mods); do \
        (cd $$(dirname $$f); go run $(golangci_lint) run --timeout 5m); \
	done

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -local sigs.k8s.io/kube-scheduler-wasm-extension/ -w $(shell find . -name '*.go' -type f)

all_examples :=  $(wildcard ./examples/*/go.mod)
# all_mods are the go modules including examples. Examples should also be
# formatted, lint checked, etc. even if they are are built with TinyGo.
all_mods     := ./internal/e2e/go.mod ./scheduler/go.mod ./guest/go.mod ./kubernetes/proto/go.mod $(all_examples)

.PHONY: tidy
tidy:
	@for f in $(all_mods); do \
        (cd $$(dirname $$f); go mod tidy); \
	done

.PHONY: build
build:
	@# We filter out the examples main packages, as nottinygo cannot compiled \
     # to a normal platform executable.
	@for f in $(filter-out $(all_examples), $(all_mods)); do \
        (cd $$(dirname $$f); go build ./...); \
	done

.PHONY: test-scheduler
test-scheduler:
	@for d in ./scheduler ./internal/e2e; do \
        (cd $$d; go test ./...); \
	done

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
