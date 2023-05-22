go_sources := ./abimain/main.go ./vfsmain/main.go
.PHONY: build
build: $(go_sources)
	@for f in $^; do \
		CGO_ENABLED=0 tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
	done
# Note: GOOS=wasip1 compiles vfsmain, but it is much slower in execution.
#	   GOOS=wasip1 GOARCH=wasm gotip build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') $$f; \
