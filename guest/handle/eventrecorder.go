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

package handle

import (
	"encoding/json"
	"runtime"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/mem"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
)

type eventRecorder struct {
	EventfFn func(msg EventMessage)
}

var eventRecorderInstance api.EventRecorder = &eventRecorder{
	EventfFn: EventfFn,
}

func EventfFn(msg EventMessage) {
	jsonByte, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	jsonStr := string(jsonByte)
	ptr, size := mem.StringToPtr(jsonStr)
	eventf(ptr, size)
	runtime.KeepAlive(jsonStr)
}

func (e *eventRecorder) Eventf(regarding internalproto.KObject, related internalproto.KObject, eventtype, reason, action, note string) {
	reg := convertToObjRef(regarding)
	rel := convertToObjRef(related)

	msg := EventMessage{
		RegardingReference: reg,
		RelatedReference:   rel,
		Eventtype:          eventtype,
		Reason:             reason,
		Action:             action,
		Note:               note,
	}
	e.EventfFn(msg)
}

func convertToObjRef(obj internalproto.KObject) ObjectReference {
	if obj == nil {
		return ObjectReference{}
	}
	objRef := ObjectReference{
		Kind:            obj.GetKind(),
		APIVersion:      obj.GetApiVersion(),
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		UID:             obj.GetUid(),
		ResourceVersion: obj.GetResourceVersion(),
	}
	return objRef
}

// Using ObjectReference because it includes necessary information for framework.handle.eventf.
type ObjectReference struct {
	Kind            string
	APIVersion      string
	Name            string
	Namespace       string
	UID             string
	ResourceVersion string
}

type EventMessage struct {
	RegardingReference ObjectReference
	RelatedReference   ObjectReference
	Eventtype          string
	Reason             string
	Action             string
	Note               string
}
