go_sources := ./vfsmain/main.go
.PHONY: build
build: $(go_sources)
	@for f in $^; do \
	    GOOS=wasip1 GOARCH=wasm gotip build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') $$f; \
	done
# TODO: TinyGo hangs compiling possibly due to https://github.com/tinygo-org/tinygo/issues/3653#issuecomment-1529847344
#	    CGO_ENABLED=0 tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
