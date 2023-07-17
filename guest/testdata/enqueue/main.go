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

// Override the default GC with a more performant one.
// Note: this requires tinygo flags: -gc=custom -tags=custommalloc
import (
	"os"

	_ "github.com/wasilibs/nottinygc"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/enqueue"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "0":
		case "1":
			clusterEvents = []api.ClusterEvent{
				{Resource: api.PersistentVolume, ActionType: api.Delete},
			}
		case "2":
			clusterEvents = []api.ClusterEvent{
				{Resource: api.Node, ActionType: api.Add},
				{Resource: api.PersistentVolume, ActionType: api.Delete},
			}
		default:
			panic("unsupported count")
		}

	}
	enqueue.SetPlugin(enqueueExtensions{})

}

var clusterEvents []api.ClusterEvent

type enqueueExtensions struct{}

func (enqueueExtensions) EventsToRegister() []api.ClusterEvent { return clusterEvents }
