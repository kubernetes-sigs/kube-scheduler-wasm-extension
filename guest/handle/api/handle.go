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
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
)

// Handle provides data and some tools that plugins can use.
//
// This contains functions like framework.Handle.
type Handle interface {
	// EventRecorder returns an event recorder.
	EventRecorder() EventRecorder
}

type EventRecorder interface {
	// Eventf calls framework.Handle.Event.
	Eventf(regarding internalproto.KObject, related internalproto.KObject, eventtype, reason, action, note string)
}
