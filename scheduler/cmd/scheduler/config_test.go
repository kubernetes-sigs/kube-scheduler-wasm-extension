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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/component-base/logs/json/register"
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
)

func Test_getWasmPluginNames(t *testing.T) {
	t.Parallel()
	type args struct {
		cc *config.KubeSchedulerConfiguration
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no wasm plugin",
			args: args{
				cc: &config.KubeSchedulerConfiguration{
					Profiles: []config.KubeSchedulerProfile{
						{
							SchedulerName: "default-scheduler",
							Plugins: &config.Plugins{
								MultiPoint: config.PluginSet{
									Enabled: []config.Plugin{
										{Name: "DefaultPreemption"},
										{Name: "InterPodAffinity"},
									},
								},
							},
							PluginConfig: []config.PluginConfig{
								{
									Name: "DefaultPreemption",
									Args: &config.DefaultPreemptionArgs{},
								},
								{
									Name: "InterPodAffinity",
									Args: &config.InterPodAffinityArgs{},
								},
							},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "wasm plugin is in the config",
			args: args{
				cc: &config.KubeSchedulerConfiguration{
					Profiles: []config.KubeSchedulerProfile{
						{
							SchedulerName: "default-scheduler",
							Plugins: &config.Plugins{
								MultiPoint: config.PluginSet{
									Enabled: []config.Plugin{
										{Name: "DefaultPreemption"},
										{Name: "InterPodAffinity"},
										{Name: "wasm1"},
									},
								},
							},
							PluginConfig: []config.PluginConfig{
								{
									Name: "DefaultPreemption",
									Args: &config.DefaultPreemptionArgs{},
								},
								{
									Name: "InterPodAffinity",
									Args: &config.InterPodAffinityArgs{},
								},
								{
									Name: "wasm1",
									Args: &runtime.Unknown{
										// TODO: need to make the wasm config implements runtime.Object.
										Raw: []byte(`{"guestURL":"https://example.com/hoge.wasm"}`),
									},
								},
								{
									// wasm2 is in the config, but not enabled.
									Name: "wasm2",
									Args: &runtime.Unknown{
										// TODO: need to make the wasm config implements runtime.Object.
										Raw: []byte(`{"guestURL":"https://example.com/hoge.wasm"}`),
									},
								},
							},
						},
					},
				},
			},
			want: []string{"wasm1"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := getWasmPluginNames(tt.args.cc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getWasmPluginNames() = %v, want %v", got, tt.want)
			}
		})
	}
}
