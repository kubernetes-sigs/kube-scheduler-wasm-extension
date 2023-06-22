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

## Why do host functions use protobuf messages?

Kubernetes Scheduler plugins need to access very large model data, specifically
node and pod data. WebAssembly has a sandbox model, so the memory of a plugin
is not the same as host scheduler process. This means the node and pod are not
passed by reference. In fact, they cannot be copied by value either. There are
two ways typically used considering this: ABI based model or a serialized one.

An ABI based model based on stable versions of WebAssembly can only use numeric
types and memory. For example, a string is a pre-defined encoding of a byte
range with validation rules on either side. At least one WebAssembly function
needs to be exported to the guest per field, to read it. If the field is
mutable, two or three other functions per field would be needed.

For example, a function to get an HTTP uri could look like this on the guest:
```webassembly
(import "http_handler" "get_uri" (func $get_uri
  (param $buf i32) (param $buf_limit i32)
  (result (; uri_len ;) i32)))
```

The host would implement that import like this:
```go
func (m *middleware) getURI(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	uri := m.host.GetURI(ctx)
	uriLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, uri)

	stack[0] = uint64(uriLen)
}
```

A small amount of stable fields (<=20) can be optimally done in an ABI model,
even manually. In stable/small case, there isn't a lot of glue code needed and
there is little maintenance to those functions over time. It is the most
performant and efficient way to communicate.

However, this doesn't match the use case of Kubernetes. The node and pod models
include several hundred data types, with nearly a thousand fields. To deal with
a model this large would require a code generator. Even if such a code
generator were sourced or made, it would require automatic conversion from the
incoming proto model used on the host, as the conversion logic would be too
large to maintain manually. Many other complications would follow suit,
including an amplified number of host calls. In summary, a usable ABI binding
would be an effort larger than the scheduler plugin itself.

The path of least resistance is a marshalling approach. Almost all parameters
of size used in scheduler plugins are generated protobuf model types. While not
performant, it is possible to have the host marshal these objects and unmarshal
them as needed on the guest. Further optimizations could be made as many of the
underlying data do not change within a plugin lifecycle.

For example, a function to get a pod could look like this on the guest:
```webassembly
(import "k8s.io/api" "pod" (func $get_pod
  (param $buf i32) (param $buf_limit i32)
  (result (; pod_len ;) i32)))
```

The host would marshal the entire pod argument to protobuf like so:
```go
func k8sApiPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pod := filterArgsFromContext(ctx).pod
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), pod, buf, bufLimit))
}
```

The main tradeoff to this approach is performance. The model is very large and
the default unmarshaller creates a lot of garbage. We have some workarounds
noted below, and also there are options for mitigation not yet implemented:

* updating heavy objects only when they change
  * this limits the occurrence count of decode performance
* tracking of alternative encoding libraries which use protobuf IDL
  * [polyglot][4] can generate code from protos and might automatically convert
    protos to its more efficient representation.

## Why do we recommend nottinygc, but not import it by default?

Unmarshalling protobuf messages with default tooling creates a lot of garbage.
As WebAssembly has only one thread, garbage collection is inlined. TinyGo's
compiler optimized for code size not speed. In the default configuration, we
found inlined GC overhead to be over half the latency of a plugin execution.

[nottinygc][5] is an alternate garbage collection implementation for TinyGo.
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
[wazero's concurrency page][6]. Hence, this is not a new constraint.

A last understood tradeoff is nottinygc is new and currently only one
contributor. That said, the lead maintainer is very responsive and the
scheduler plugin itself is early.

Considering the above, we recommend nottinygc, but as an opt-in process. Our
examples default to configure it, and all our integration tests use it.
However, we can't make this default until it no longer crashes our unit tests.

[1]: https://tinygo.org/
[2]: https://pkg.go.dev/golang.org/dl/gotip
[3]: https://github.com/golang/go/issues/42372
[4]: https://github.com/loopholelabs/polyglot-go
[5]: https://github.com/wasilibs/nottinygc
[6]: https://wazero.io/languages/tinygo/#concurrency