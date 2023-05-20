go_sources := ./vfsmain/main.go
.PHONY: build
build: $(go_sources)
	@for f in $^; do \
	    tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
	done
# TODO: TinyGo 0.27 can't compile "k8s.io/api v0.26.2" You have to use gotip, but it is very slow
#	    GOOS=wasip1 GOARCH=wasm gotip build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') $$f; \
