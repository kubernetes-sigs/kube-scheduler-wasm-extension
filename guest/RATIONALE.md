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

Plugins can implement multiple hooks(i.e. extension points in scheduler framework), such as `PreFilter` and `Filter`. An FFI
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

## Why do we compile with the `target=wasi`?

TinyGo only has JavaScript and WASI targets. It has no [freestanding one][6].
Kubernetes can't in a browser, so the only choice is '-target=wasi'. So, we use
this despite the ABI (application binary interface) of the scheduler not
requiring it at all.

TinyGo imports the "wasi_snapshot_preview1" module for [runtime][7] concerns
like args, stdio and random number generation. The technical implementation is
defined in wazero's built-in [wasi_snapshot_preview1][8] package, which the
host side of the scheduler plugin conditionally configures.

## Why are plugins assigned with functions instead of global variables?

At first, we designed the guest SDK to assign plugins with global variables.
For example, `prefilter.Plugin = api.PreFilterFunc(podSpecName)`. This worked
until we needed to control cycle state.

When we moved cycle state to an internal package, we had to move the `Plugin`
field also, to avoid package cycles. In Go, we cannot map a global variable in
one package to another, such that setting one sets the other. The simplest way
out was to change to a function instead.

```diff
 func main() {
-       prefilter.Plugin = api.PreFilterFunc(podSpecName)
+       prefilter.SetPlugin(api.PreFilterFunc(podSpecName))
 }
```

## Why does `CycleState.Read` return a boolean instead of an error.

In the platform framework, `CycleState.Read` returns a value and an error, but
an error is only defined in the case of a value missing. This returns a boolean
instead as it is a more common and efficient way to represent key not found.

## Why do we recommend nottinygc, but not import it by default?

Unmarshalling protobuf messages with default tooling creates a lot of garbage.
As WebAssembly has only one thread, garbage collection is inlined. TinyGo's
compiler optimized for code size not speed. In the default configuration, we
found inlined GC overhead to be over half the latency of a plugin execution.

[nottinygc][4] is an alternate garbage collection implementation for TinyGo.
This optimizes for performance instead of bytecode size. Swapping this adds
110KB to the base size of the wasm guest, but results in a 48pct drop in
execution time of a simple plugin, in a scale of hundreds of microseconds.

nottinygc has tradeoffs besides size. One is that it isn't built-in to TinyGo.
To use nottinygc requires custom flags in the build process, in addition to the
custom ones we already have. Running unit tests via (`tinygo test`) against
packages that import nottinygc have resulted in segfaults during GC.

nottinygc also requires a flag `-scheduler=none`, which means end users can't
use tools like asyncify. However, they can't anyway. Scheduler plugin functions
are implemented as custom function exports, while TinyGo only supports
goroutines inside `main`. Elsewhere, causes a runtime panic, as noted on
[wazero's concurrency page][5]. Hence, this is not a new constraint.

A last understood tradeoff is nottinygc is new and currently only one
contributor. That said, the lead maintainer is very responsive and the
scheduler plugin itself is early.

Considering the above, we recommend nottinygc, but as an opt-in process. Our
examples default to configure it, and all our integration tests use it.
However, we can't make this default until it no longer crashes our unit tests.

## Why don't we use the normal k8s.io/klog/v2 package for logging?

The scheduler framework uses the k8s.io/klog/v2 package for logging, like:
```go
klog.InfoS("execute Score on NodeNumber plugin", "pod", klog.KObj(pod))
```

The guest SDK cannot use this because some parts of the klog package do not
compile with TinyGo, due to heavy use of reflection. Also, the initialization
of the wasm guest is separate from the scheduler process, and it wouldn't be
able to read the same configuration including filters that need to be applied.

Instead, this adds a minimal working abstraction of klog functions which pass
strings to the host to log using a real klog function.

As discussed in other sections, you cannot pass an object between the guest and
the host by reference, rather only by value. For this reason, the guest klog
package stringifies args including key/values and sends them to the host for
processing via functions like `klog.Info` or `klog.ErrorS`.

Stringification is expensive in Wasm due to factors including inlined garbage
collection. To avoid performance problems when not logging, the host includes a
function not in the normal `klog` package, which exposes the current severity
level. Anything outside that level won't be logged, and that's how excess
overhead is avoided.

### Why does `klog.KObj` return a `fmt.Stringer` instead of `ObjectRef`

`klog.KObj` works differently in wasm because unlike the normal scheduler
framework, objects such as `proto.Node` are lazy unmarshalled. To avoid
triggering this when logging is disabled, `klog.KObj` returns a `fmt.Stringer`
which lazy accessed fields needed.

### Why is there an `api` package in `klog`?

The `klog` package imports functions from the host, via `//go:wasmimport`. This
makes code in that package untestable via `tinygo test -target=wasi`, as the
implicit Wasm runtime launched does not export these (they are custom to this
codebase). To allow unit testing of the core logic with both Go and TinyGo, we
have an `api` package which includes an interface representing the logging
functions in the `klog` package. Advanced users can also use these interfaces
for the same reason: to keep their core logic testable in TinyGo.

[1]: https://tinygo.org/
[2]: https://pkg.go.dev/golang.org/dl/gotip
[3]: https://github.com/golang/go/issues/42372
[4]: https://github.com/wasilibs/nottinygc
[5]: https://wazero.io/languages/tinygo/#concurrency
[6]: https://github.com/tinygo-org/tinygo/pull/3072
[7]: https://github.com/tinygo-org/tinygo/blob/v0.28.1/src/runtime/runtime_wasm_wasi.go
[8]: https://pkg.go.dev/github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1