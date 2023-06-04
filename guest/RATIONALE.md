# Go Guest SDK

This package is a Go programming SDK for the guest side of the scheduler
plugin. This document describes rationale of notable or less intuitive
decisions in design or implementation.

## Why do we use TinyGo instead of normal Go?

This project began in mid-2023. Its first possible introduction in practice
would be v1.29.0 which would be in late 2023. This would be after the release
of Go 1.21 which supports compiling out-of-browser wasm via
`GOOS=wasip1 GOARCH=wasm`. A common question then is, why does this SDK use
[TinyGo][1] instead of 1.21 (via [gotip][2] until betas are out)? The main two
reasons are performance and lack of ability to export functions.

### Lack of exported functions

Plugins can implement multiple hooks, such as `PreFilter` and `Filter`. An FFI
approach to implementing plugins would export a WebAssembly function for each
hook. A sub-process (WASI) approach would use a `main` argument for each hook.

TinyGo supports the FFI approach using its `//export` directive, and when 0.28
is released, the emerging `//go:wasmexport` for the same thing. For example,
the scheduler could call a wasm function named `pre_filter` then `filter`. Go
1.21 does not plan to support [`//go:wasmexport`][3], yet, as this is deferred
to the subsequent version, which would happen in early 2024.

Both Go and TinyGo support invocation via main, except this has some
performance penalties. For example, the go runtime and memory needs to be
recreated per request. This eliminates the ability to cache state at the plugin
lifecycle, because the memory is destroyed each time.

Memory being unsharable isn't an issue for a single hook, but with multiple
hooks introduces state management problems. However, the larger issue is
described below.

### Performance

As described above, the only way to support Go 1.21 would be via the subprocess
model. However, the performance of that is extremely slow, especially vs TinyGo.
We found re-creating an instance per hook costs 45us in TinyGo, yet over 4ms in
Go. This is without doing any protocol buffer based work. Decoding even a
simple message in Go 1.21 was over 10ms.

Considering you cannot share memory in subprocess invocation style, this could
mean amplifying that overhead to a point where second or longer latency would
be possible. For reasons like this, we dismissed Go 1.21 for TinyGo, and will
revisit when Go 1.22 introduces [`//go:wasmexport`][3].


[1]: https://tinygo.org/
[2]: https://pkg.go.dev/golang.org/dl/gotip
[3]: https://github.com/golang/go/issues/42372
