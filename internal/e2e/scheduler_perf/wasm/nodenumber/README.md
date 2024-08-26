# NodeNumber Plugin

This is the nodenumber example wasm plugin, which only implements PreScore and Score.
It doesn't use any additional host functions (klog, handle, etc) so that scheduler_perf can measure the overhead truely.
