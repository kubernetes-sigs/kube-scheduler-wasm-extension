# Notable rationale of the WebAssembly ABI

## Why do some stack values (parameters and results) use protobuf encoding?

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
    * [polyglot][3] can generate code from protos and might automatically convert
      protos to its more efficient representation.

## How is `framework.CycleState` implemented in WebAssembly?

`framework.CycleState` is a primarily a key value storage. Plugins store values
in `PreFilter` or `PreScore` for use afterward. This is similar to Go context,
except the keys must be strings and the values must implement `StateData`.

While one instance of `framework.CycleState` is re-used for all plugins in a
scheduling cycle, in practice, `StateData` are not shared. For this reason, we
implement cycle state in the guest. The only responsibility of the wasm guest
is to reset any state when `PreFilter` is called. All state is invisible from
the perspective of the host.

### Why isn't `StateData.Clone` handled in WebAssembly?

`Clone` is a special case for preemption. When all Nodes are filtered out, the
scheduler attempts to make space via `PostFilter`. Preemption results in a
[parallel][4] call to `SelectVictimsOnNode` which runs in [parallel][5], and
removes victims (pods) from `PreFilter` state and `NodeInfo` before calling
[`RunFilterPluginsWithNominatedPods`][6]. If `StateData` wasn't cloneable, the
original `StateData` values would be lost.

The current WebAssembly implementation skips `Clone`, for now. Since the
guest holds the cycle state, the most likely path forward would be to export
a guest function to save the cycle state and return a state ID to restore it
with. This function could be called when the host side knows it is in the
process of preemption. This would be the case on any of the following:

* `Filter` is called a second time (before `PreFilter`)
* `AddNode` or `DeleteNode` are called

## Why are some return values different between Go and Wasm?

Bear in mind that the scheduler framework was not initially designed for remote
or embedded use, with an IDL or ABI in mind. Sometimes, results were not
consciously defined according to its bit size (e.g. `int` in Go). Other times,
results were accidentally defined very large (e.g. `int64` in Go). In many
cases, results were defined as signed without any semantic meaning for this.

This plugin cannot take the responsibility of all decisions made in the
codebase prior to it. However, we will briefly explain some pragmatic choices,
which allow this plugin to return up to two results at a time in a WebAssembly
`i64` result.

The general approach is to attempt using `i32` instead, so that a second result
can be packed into an `i64`. This is compatible with all language compilers,
because the largest return value permitted by WebAssembly Core Specification
1.0 (REC) is `i64`. Further rationale about alternate choices are discussed in
separate sections. This focuses on result value mapping.

### Status

The `framework.Status` type is used as a nilable result with a status code and
a string reason when that code is non-zero. So, zero is success and there are
also five exceptional [built-in codes][1].

Only built-in codes drive behavior: anything outside that range is treated
generically with side effects limited to logging and metrics.

#### Why is the code int32 not int (64-bit)?

The code was declared as an `int`, which a careful reader will notice is both
signed and also 64-bits wide. This was not a conscious decision that plugin
authors require billions of different status codes, rather a lack of decision.

When porting to WebAssembly, we reduce this to `i32`, a still very large
status code range, to permit another `i32` result to be returned with it.

Narrowing should not be a problem in practice because not only are no custom
status codes differentiable from a generic error, but also there are no custom
status codes known to be used as of mid 2023.

Even if there were custom status codes in the high billion range, the status
is a result, not a parameter. The only impact would be to new plugins or those
ported to WebAssembly. We do not expect limiting the range of status codes to
be over two billion to be a practical concern for these authors.

#### Why is the reason defined as a host callback?

The status reason is an optional message set when the code is non-zero. The
typical way to handle a Go `error` is to set the status reason to the string
value of the error. The string value of an error is not definable and could be
quite large in the case of an embedded stack trace. In other words, it is
variable length without any constraint except it is smaller than guest memory.

As mentioned in the overview section, WebAssembly 1.0 can return up to one
numeric result, of maximum size `i64`. As mentioned above we are already using
32 bits of that for the status code.

The only way we could pass the status reason in the remaining 32-bits would be
via a NUL-terminated string, a.k.a. CString. However, this presents three
problems: First, even on success, this would make it impossible to return
another value such as a score. Second, not all languages naturally declare
strings as CStrings, meaning some would have to add a NUL character manually.
Finally, this introduces complications in garbage collection, as the reason
string would need to be leaked to the caller, without a defined way to free it.

To allow the guest to be in charge of string allocation, the simplest and most
portable way to handle this, is to add a host callback for the status reason.

```webassembly
(func (import "k8s.io/scheduler" "status_reason")
  (param $buf i32) (param $buf_len i32))
```

Doing this only adds host call overhead on error, and it will not likely fail,
unless there is a severe programming error in the guest SDK. The host can read
the entire guest memory, so it should not have an issue with a potentially
large reason result.

### Why is score i32 not i64?

`framework.ScorePlugin` currently declares score as int64, but it is only used
as a positive number, and it is normalized to the range [[0, 100]][2].

Even if there were custom scores in the high billion range, the score is a
result, not a parameter. The only impact would be to new plugins or those
ported to WebAssembly. We do not expect limiting the scores to two billion
above the valid range to be a practical concern for these authors.

## Why do we return a non-status, second numeric result as an i32?

Most compilers that target WebAssembly Core Specification 1.0, the only REC
status specification in w3c. This allows up to one result of up to 64bits.

As described in sections above, it is commonly the case for integer result
types to be unnecessarily defined in Go as 64-bit types, while their value
range is far less than 32-bits in practice.

We currently use 32-bits to return the status as an `i32`. So, we could pass a
second 32-bit value by packing it with the result into an `i64`. This is the
approach used and detailed in this section.

When the combined results require more than 64 bits, we have to share memory to
return more results, as out pointer parameters. This is complicated when the
function is called host->guest, because the guest controls memory. For the host
to pass an out-pointer, it needs a region of memory unused by the guest.

Most often, this is reserved with exported functions. For example, TinyGo
currently exports `malloc`, and `free` by default. The host can reserve the
amount of space to write (in this case 4 bytes for an `i64`) with `malloc`, which
returns a pointer. When finished, it calls `free` to release it. For something
of fixed width, this could region could be allocated directly with the guest,
and for all function calls. This would reduce the overhead from 2*calls to
2*guest. The main disadvantage to this approach is it requires all users to
compile their wasm in a way that exports these functions. While we have only
one SDK (TinyGo), this would work, but we would need a different approach for
Go which doesn't export these functions.

Another way would be to pass another result by host callback which accepts a
single `i64` value. In this case, implementors of `score` must know to call
another function like `set_score` before returning the status. This adds both
complexity and overhead to SDK authors: First, it creates a function dependency
which needs to be implemented and explained. Then, it creates more overhead as
the guest has to hop to the host before completing, even on success.

Yet another way is to know the underlying toolchain of the compiler and use
memory the guest won't use. Two approaches to this are sneaking space from the
heap (unsafe as choosing a high value might not be free) or from the stack
(mostly safe but toolchain specific). Especially borrowing space from the stack
could be safe and cheap. For example, if you know the underlying implementation
is LLVM, you could move temporarily move the stack pointer closer to zero, and
pas the resulting gap to the guest as an out pointer. Many compilers use LLVM,
so it might seem this is "good enough". However, an ABI made for multiple
languages will be limited to compilers who have the same implementation. This
plugin has a goal to later with normal Go, which doesn't use LLVM and doesn't
export its stack pointer for manipulation either. So, we cannot use this way.

Yet even another way would be to define an initialization callback with scratch
space for fixed-width, secondary results. The guest would allocate and hold a
region for use in the host, for example, by declaring a field with a byte array
it doesn't use. It would return the pointer to the start of this buffer to the
host. This would be safe because it won't be garbage collected by the guest.
It is also not toolchain specific, as any language can implement this
explicitly. There's no risk of clashed usage of this field because the host
always uses the guest module sequentially. As long as the host only uses this
as scratch space, e.g. never leaks a pointer inside this region, it is safe.
The main downside of this approach is that it a more complicated and expensive
approach than using a stack return value.

In summary, the current approach is to attempt map a second result type as an
`i32` and pack it with the status code as a single `i64` return. This avoids
complexity for now. If later, we end up with more complicated, or more than two
non-status results, we may revisit this decision and possibly setup scratch
space instead.

[1]: https://github.com/kubernetes/kubernetes/blob/v1.27.3/pkg/scheduler/framework/interface.go#L79-L99
[2]: https://github.com/kubernetes/kubernetes/blob/v1.27.3/pkg/scheduler/framework/interface.go#L110-L114
[3]: https://github.com/loopholelabs/polyglot-go
[4]: https://github.com/kubernetes/kubernetes/blob/v1.27.3/pkg/scheduler/framework/preemption/preemption.go#L606
[5]: https://github.com/kubernetes/kubernetes/blob/v1.27.3/pkg/scheduler/plugins/defaultpreemption/default_preemption.go#L139
[6]: https://github.com/kubernetes/kubernetes/blob/v1.27.3/pkg/scheduler/framework/runtime/framework.go#L826-L827
