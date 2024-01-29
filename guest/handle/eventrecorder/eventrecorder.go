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

package eventrecorder

import (
	"encoding/json"
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/eventrecorder/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/eventrecorder/internal"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
)

func Get() api.EventRecorder {
	return eventRecorderInstance
}

var eventRecorderInstance api.EventRecorder = &internal.EventRecorder{
	EventfFn: EventfFn,
}

func EventfFn(msg internal.EventMessage) {
	jsonByte, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	jsonStr := string(jsonByte)
	ptr, size := mem.StringToPtr(jsonStr)
	eventf(ptr, size)
	runtime.KeepAlive(jsonStr)
}

// Eventf is a convenience that calls the same method documented on api.Eventf.
//
// Note: See Info for unit test and benchmarking impact.
func Eventf(regarding proto.KObject, related proto.KObject, eventtype, reason, action, note string) {
	eventRecorderInstance.Eventf(regarding, related, eventtype, reason, action, note)
}
