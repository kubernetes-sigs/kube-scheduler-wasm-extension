## Examples

This examples help you understand how you build your wasm plugins with our Go SDK (which is the only language SDK supported for now).

- [NodeNumber Plugin](./nodenumber/): The simple example plugin in which you can simply get how the wasm plugin looks like.
- [Adbanced NodeNumber Plugin](./advanced/): The example plugin one step advanced, which is more complicated than the first one, but more efficient and testable.

### Go SDK Interfaces

Like [Scheduling Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/), the Go SDK provides [the interfaces](../guest/api/types.go).

Some of them look different from [the interfaces in Scheduling Framework](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go) though, it's the same how they're called by the scheduler.
