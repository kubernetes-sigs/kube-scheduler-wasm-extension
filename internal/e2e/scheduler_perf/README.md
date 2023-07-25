## What's this

It's the benchmark to compare three extension ways:
- Plugins (Scheduling Framework)
- Extenders
- Wasm

It's created based on [scheduler_perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf).

## How to run

```
# run the benchmark.
go test -run=^$ -benchtime=1ns -bench=BenchmarkPerfScheduling
```
