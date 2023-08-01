# Advanced NodeNumber Plugin

This is a WebAssembly port of the [Scheduler Simulator NodeNumber plugin][1].

This variant is more complicated than the [simple plugin][2] to program, but
more efficient and testable.

* This manually configures lifecycle hooks to avoid no-op overhead.
* This uses the more efficient [nottinygc][3] garbage collector.
* The `plugin` package can be tested both with `go test` and
  `tinygo test -target=wasi`.
* See [RATIONALE.md](../../guest/RATIONALE.md) for more notes on performance.

[1]: https://github.com/kubernetes-sigs/kube-scheduler-simulator/blob/simulator/v0.1.0/simulator/docs/sample/nodenumber/plugin.go

[2]: ../nodenumber

[3]: https://github.com/wasilibs/nottinygc