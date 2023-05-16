# kube-scheduler-wasm-extension

All the things to make the scheduler extendable with [wasm](https://webassembly.org/).

This project is composed of:
- The SDK to build your wasm-ed plugin.
- The scheduler plugin to load your wasm-ed plugin.

## Motivation

Nowadays, the scheduler can be extended via 
- [recommended] add custom scheduler plugins based on [Scheduling Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).
- configure [Extenders](https://github.com/kubernetes/design-proposals-archive/blob/main/scheduling/scheduler_extender.md)

Although building scheduler plugins is the recommended way to extend,
it requires non-trivial works: 
When you want to integrate your plugins into scheduler,
you need to re-build the scheduler with your plugins, replace the vanilla scheduler with it, 
and keep doing them whenever you want to upgrade your cluster.

In this project, we aim that users can use their own plugins by giving wasm binary to the scheduler,
which frees users from the above tasks.

And furthermore, users write their own scheduler plugins in any languages other than Golang as well.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
