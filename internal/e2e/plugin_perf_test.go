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

package e2e_test

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	wasm "sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/plugin"
	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

func BenchmarkPluginFilter(b *testing.B) {
	ctx := context.Background()

	plugins, close := newTestPlugins(b, ctx, wasm.PluginConfig{GuestPath: test.PathTestFilter})
	defer close()

	tests := []struct {
		name string
		pod  *v1.Pod
		node *v1.Node
	}{
		{
			name: "params: small",
			pod:  test.PodSmall,
			node: test.NodeSmall,
		},
		{
			name: "params: real",
			pod:  test.PodReal,
			node: test.NodeReal,
		},
	}

	for _, tp := range plugins {
		pl := tp
		b.Run(pl.name, func(b *testing.B) {
			for _, tc := range tests {
				b.Run(tc.name, func(b *testing.B) {
					ni := framework.NewNodeInfo()
					ni.SetNode(tc.node)

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						s := pl.plugin.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)
						if want, have := framework.Success, s.Code(); want != have {
							b.Fatalf("unexpected code: got %v, expected %v, got reason: %v", want, have, s.Message())
						}
					}
				})
			}
		})
	}
}

func BenchmarkPluginScore(b *testing.B) {
	ctx := context.Background()

	plugins, close := newTestPlugins(b, ctx, wasm.PluginConfig{GuestPath: test.PathTestScore})
	defer close()

	tests := []struct {
		name string
		pod  *v1.Pod
	}{
		{
			name: "params: small",
			pod:  test.PodSmall,
		},
		{
			name: "params: real",
			pod:  test.PodReal,
		},
	}

	for _, tp := range plugins {
		pl := tp
		b.Run(pl.name, func(b *testing.B) {
			for _, tc := range tests {
				b.Run(tc.name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, s := pl.plugin.(framework.ScorePlugin).Score(ctx, nil, tc.pod, tc.pod.Spec.NodeName)
						if want, have := framework.Success, s.Code(); want != have {
							b.Fatalf("unexpected status code: got %v, expected %v, got reason: %v", want, have, s.Message())
						}
					}
				})
			}
		})
	}
}

func BenchmarkPluginFilterAndScore(b *testing.B) {
	ctx := context.Background()

	plugins, close := newTestPlugins(b, ctx, wasm.PluginConfig{GuestPath: test.PathTestAll})
	defer close()

	tests := []struct {
		name string
		pod  *v1.Pod
		node *v1.Node
	}{
		{
			name: "params: small",
			pod:  test.PodSmall,
			node: test.NodeSmall,
		},
		{
			name: "params: real",
			pod:  test.PodReal,
			node: test.NodeReal,
		},
	}

	for _, tp := range plugins {
		pl := tp
		b.Run(pl.name, func(b *testing.B) {
			for _, tc := range tests {
				b.Run(tc.name, func(b *testing.B) {
					ni := framework.NewNodeInfo()
					ni.SetNode(tc.node)

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						s := pl.plugin.(framework.FilterPlugin).Filter(ctx, nil, tc.pod, ni)
						if want, have := framework.Success, s.Code(); want != have {
							b.Fatalf("unexpected code: got %v, expected %v, got reason: %v", want, have, s.Message())
						}
						_, s = pl.plugin.(framework.ScorePlugin).Score(ctx, nil, tc.pod, tc.pod.Spec.NodeName)
						if want, have := framework.Success, s.Code(); want != have {
							b.Fatalf("unexpected status code: got %v, expected %v, got reason: %v", want, have, s.Message())
						}
					}
				})
			}
		})
	}
}

func newTestPlugins(b *testing.B, ctx context.Context, config wasm.PluginConfig) ([]struct {
	name   string
	plugin framework.Plugin
}, func(),
) {
	noopTinyGo, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: test.PathTestAllNoopTinyGo})
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}

	noopWat, err := wasm.NewFromConfig(ctx, wasm.PluginConfig{GuestPath: test.PathTestAllNoopWat})
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}

	plugin, err := wasm.NewFromConfig(ctx, config)
	if err != nil {
		b.Fatalf("failed to create plugin: %v", err)
	}

	return []struct {
			name   string
			plugin framework.Plugin
		}{
			{
				name:   "noop-wat", // absolute base case
				plugin: noopWat,
			},
			{
				name:   "noop", // base case for TinyGo
				plugin: noopTinyGo,
			},
			{
				name:   "test",
				plugin: plugin,
			},
		}, func() {
			_ = noopWat.Close()
			_ = noopTinyGo.Close()
			_ = plugin.Close()
		}
}
