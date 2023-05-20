# kube-scheduler-wasm-extension

[WebAssembly](https://webassembly.org/) is a way to safely run code compiled in other
languages. Runtimes execute WebAssembly Modules (Wasm), which are most often
binaries with a `.wasm` extension.
This project allows you to extend the kube-scheduler with custom scheduler plugin compiled to a Wasm
binary. It works by embedding a WebAssembly runtime, [wazero](https://wazero.io), into the
scheduler, and loading custom scheduler plugin via configuration.

This project contains everything needed to extend the scheduler:
- Documentation describing what type of actions are possible, e.g. `Filter`.
- Language SDKs used to build scheduler plugins, compiled to wasm.
- The scheduler plugin which loads and runs wasm plugins

## Motivation

Nowadays, the kube-scheduler can be extended via [Scheduling Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).
But, it requires non-trivial works: 
When you want to integrate your plugins into scheduler,
you need to re-build the scheduler with your plugins, replace the vanilla scheduler with it, 
and keep doing them whenever you want to upgrade your cluster.

In this project, we aim that users can use their own plugins by giving wasm binary to the scheduler,
which frees users from the above tasks.

Furthermore, programming language is no longer the problem anymore, you can use whatever programming language you like to build your own scheduler plugin.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
