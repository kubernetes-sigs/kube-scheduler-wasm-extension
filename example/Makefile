.PHONY: build
build: main.go
	@for f in $^; do \
		CGO_ENABLED=0 tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
	done
