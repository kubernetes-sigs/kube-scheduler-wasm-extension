package main

import (
	"os"

	"k8s.io/component-base/cli"
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // for rest client metric registration
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/pkg/plugins/wasm"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(wasm.PluginName, wasm.New),
	)

	code := cli.Run(command)
	os.Exit(code)
}
