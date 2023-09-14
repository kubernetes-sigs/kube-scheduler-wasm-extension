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

package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/sets"
	_ "k8s.io/component-base/logs/json/register" // for JSON log format registration
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version" // for version metric registration
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"

	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
)

// getWasmPluginsFromConfig parses the scheduler configuration specified with --config option,
// and return the wasm plugins enabled by the user.
func getWasmPluginsFromConfig() ([]string, error) {
	// In the scheduler, the path to the scheduler configuration is specified with --config option.
	configFile := flag.String("config", "", "")
	flag.Parse()

	if configFile == nil {
		// Users don't have the own configuration. do nothing.
		return nil, nil
	}

	cfg, err := loadConfigFromFile(*configFile)
	if err != nil {
		return nil, err
	}

	return getWasmPluginNames(cfg), nil
}

// getWasmPluginNames returns the wasm plugin names enabled by the user.
// It assumes that the wasm plugin is specified as the multi-point plugin.
func getWasmPluginNames(cc *config.KubeSchedulerConfiguration) []string {
	names := []string{}
	for _, profile := range cc.Profiles {
		wasmplugins := sets.New[string]()
		// look for the wasm plugin in the plugin config.
		for _, config := range profile.PluginConfig {
			if err := frameworkruntime.DecodeInto(config.Args, &wasm.PluginConfig{}); err != nil {
				// not wasm plugin.
				continue
			}

			wasmplugins.Insert(config.Name)
		}

		// look for the wasm plugin in the enabled plugins.
		// (assuming that the wasm plugin is specified as a multi-point plugin.)
		for _, plugin := range profile.Plugins.MultiPoint.Enabled {
			if wasmplugins.Has(plugin.Name) {
				names = append(names, plugin.Name)
			}
		}
	}

	return names
}

func loadConfigFromFile(path string) (*config.KubeSchedulerConfiguration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// The UniversalDecoder runs defaulting and returns the internal type by default.
	obj, gvk, err := scheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	cfgObj, ok := obj.(*config.KubeSchedulerConfiguration)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %v", gvk)
	}
	return cfgObj, nil
}
