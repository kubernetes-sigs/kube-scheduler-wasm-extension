# NodeNumber Plugin

This is a WebAssembly port of the [Scheduler Simulator NodeNumber plugin][1].

## Performance

This example was made to be simple to program and test with Go, but there are
some tradeoffs: This cannot be tested with TinyGo, and has higher runtime
overhead than more [advanced][2] approaches.

[1]: https://github.com/kubernetes-sigs/kube-scheduler-simulator/blob/simulator/v0.1.0/simulator/docs/sample/nodenumber/plugin.go

[2]: ../advanced
