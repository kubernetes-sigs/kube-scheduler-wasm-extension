# kube-scheduler-wasm-extension

[WebAssembly](https://webassembly.org/) is a way to safely run code compiled in other
languages. Runtimes execute WebAssembly Modules (Wasm), which are most often
binaries with a `.wasm` extension.
This project allows you to extend the kube-scheduler with custom scheduler plugin compiled to a Wasm
binary. It works by embedding a WebAssembly runtime, [wazero](https://wazero.io), into the
scheduler, and loading custom scheduler plugin via configuration.

This project contains everything needed to extend the scheduler:
- Documentation describing what type of actions are possible, e.g. `Filter`.
- [Language SDKs](./guest/) used to build scheduler plugins, compiled to wasm.
- [The scheduler plugin](./scheduler/) which loads and runs wasm plugins

You can learn how you build your wasm plugins by referring to [the tutorial](./docs/tutorial.md).

## Motivation

Today, you can extend the kube-scheduler with [Scheduling Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/), 
but it is non-trivial. 
To customize the scheduler means writing Go, and a complicated build process. 
Once you've built your scheduler, you have deployment and configuration work to have your cluster use it. 
This isn't one time work, as it needs to be redone on every upgrade of your cluster.

This project lowers that tension by making the default scheduler capable of loading custom plugins, compiled to WebAssembly (wasm). 
This removes the deployment burdens above. It also allows plugins to be written in languages besides Go.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
