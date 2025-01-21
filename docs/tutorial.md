## Tutorial

This is the basic tutorial describing the basic flow you can follow to build your scheduler plugins made of wasm.

### Build your wasm binary via SDK

[Go SDK](../guest/) is the only language SDK that we support for now.

Like [Scheduling Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/), 
the Go SDK provides [the interfaces](../guest/api/types.go) so that you can develop your own scheduling 
via a similar experience to Go scheduler plugin.

Some of them look different from [the interfaces in Scheduling Framework](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go) though, 
it's the same how they're called by the scheduler.

You can learn how you can implement your wasm plugins by referring to [examples](../examples/).

Currently, Go SDK uses TinyGo for its compile, you can refer to [Makefile](../Makefile) to see how we build example implementations.
Each example takes a different approach to build them, as README in them describes.

### Integrate your wasm binary into your scheduler

We have [a scheduler plugin](../scheduler/) which loads and runs wasm plugins.
For now, you have to build [the scheduler with the plugin](../scheduler/cmd/scheduler/main.go) by yourself though, 
we'll provide an official docker image after our first release.

You have to enable the scheduler plugin with your wasm binary in `KubeSchedulerConfiguration`, as following.

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - plugins:
      multiPoint:
        enabled:
           - name: wasmplugin1
           - name: wasmplugin2
      pluginConfig:
       - name: wasmplugin1
          args:
            guestURL: "file://path/to/wasm-plugin1.wasm"
       - name: wasmplugin2
          args:
            guestURL: "https://url/to/wasm-plugin2.wasm"
```

- A wasm plugin **must** be enabled via `multiPoint` even if your wasm plugin only uses some of extension points.
- All plugins with the plugin config matching [the wasm config format](../scheduler/plugin/config.go) are considered to be wasm plugins. 
