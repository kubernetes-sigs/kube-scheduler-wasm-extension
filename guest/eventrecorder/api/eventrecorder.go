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

package api

import (
	proto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
)

// EventRecorder is a recorder that sends events to the host environment via eventrecorder.eventf.
type EventRecorder interface {
	// Eventf calls EventRecorder.Eventf.
	Eventf(regarding proto.KObject, related proto.KObject, eventtype, reason, action, note string)
}

type UnimplementedEventRecorder struct{}

// Eventf implements events.EventRecorder.Eventf()
func (UnimplementedEventRecorder) Eventf(regarding proto.KObject, related proto.KObject, eventtype, reason, action, note string) {
}
