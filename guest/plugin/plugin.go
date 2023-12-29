/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package plugin

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/bind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/filter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle"
	handleapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/prefilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postbind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/postfilter"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prebind"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/prescore"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/reserve"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/scoreextensions"
)

// Set is a convenience to assign lifecycle hooks based on which
// interfaces `plugin` defines.
//
//	func main() {
//		plugin.Set(func(h handleapi.Handle) api.Plugin {return myPlugin{} })
//	}
//
// Note: Using this results in the host call this plugin for every hook, even
// when it isn't implemented. For more control and performance, set each hook
// individually:
//
//	func main() {
//		plugin := myPlugin{}
//		prefilter.SetPlugin(func(h handleapi.Handle) api.PreFilterPlugin { return plugin })
//		filter.SetPlugin(func(h handleapi.Handle) api.FilterPlugin { return plugin })
//	}
func Set(pluginInitializer func(h handleapi.Handle) api.Plugin) {
	handle := handle.NewFrameworkHandle()
	plugin := pluginInitializer(handle)

	if plugin, ok := plugin.(api.EnqueueExtensions); ok {
		enqueue.SetPlugin(func(h handleapi.Handle) api.EnqueueExtensions { return plugin })
	}
	if plugin, ok := plugin.(api.PreFilterPlugin); ok {
		prefilter.SetPlugin(func(h handleapi.Handle) api.PreFilterPlugin { return plugin })
	}
	if plugin, ok := plugin.(api.FilterPlugin); ok {
		filter.SetPlugin(func(h handleapi.Handle) api.FilterPlugin { return plugin })
	}
	if plugin, ok := plugin.(api.PostFilterPlugin); ok {
		postfilter.SetPlugin(func(h handleapi.Handle) api.PostFilterPlugin { return plugin })
	}
	if plugin, ok := plugin.(api.PreScorePlugin); ok {
		prescore.SetPlugin(func(h handleapi.Handle) api.PreScorePlugin { return plugin })
	}
	if plugin, ok := plugin.(api.ScorePlugin); ok {
		score.SetPlugin(func(h handleapi.Handle) api.ScorePlugin { return plugin })
	}
	if plugin, ok := plugin.(api.ScoreExtensions); ok {
		scoreextensions.SetPlugin(func(h handleapi.Handle) api.ScoreExtensions { return plugin })
	}
	if plugin, ok := plugin.(api.ReservePlugin); ok {
		reserve.SetPlugin(func(h handleapi.Handle) api.ReservePlugin { return plugin })
	}
	if plugin, ok := plugin.(api.PreBindPlugin); ok {
		prebind.SetPlugin(func(h handleapi.Handle) api.PreBindPlugin { return plugin })
	}
	if plugin, ok := plugin.(api.BindPlugin); ok {
		bind.SetPlugin(func(h handleapi.Handle) api.BindPlugin { return plugin })
	}
	if plugin, ok := plugin.(api.PostBindPlugin); ok {
		postbind.SetPlugin(func(h handleapi.Handle) api.PostBindPlugin { return plugin })
	}
}
