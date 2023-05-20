go_sources := ./vfsmain/main.go
.PHONY: build
build: $(go_sources)
	@for f in $^; do \
	    GOOS=wasip1 GOARCH=wasm gotip build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') $$f; \
	done
# TODO: TinyGo 0.27 can't compile "k8s.io/api v0.26.2" You have to use gotip, but it is very slow
#	    tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
#
# results in
# os/user
# ../../../../go/go1.20.2/go1.20.4/src/os/user/cgo_lookup_cgo.go:14:6: not implemented: build constraints in #cgo line
# ../../../../go/go1.20.2/go1.20.4/src/os/user/cgo_lookup_cgo.go:18:10: fatal: 'pwd.h' file not found
# ../../../../go/go1.20.2/go1.20.4/src/os/user/getgrouplist_unix.go:12:10: fatal: 'grp.h' file not found
