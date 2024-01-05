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

package internal

import (
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/eventrecorder/api"
)

func TestEventf(t *testing.T) {
	tests := []struct {
		name        string
		input       func(api.EventRecorder)
		expectedMsg EventMessage
	}{
		{
			name: "adds newline",
			input: func(eventrecorder api.EventRecorder) {
				eventrecorder.Eventf(podSmall{}, podSmall{}, "event", "reason", "action", "note")
			},
			expectedMsg: EventMessage{
				RegardingReference: convertToObjRef(podSmall{}),
				RelatedReference:   convertToObjRef(podSmall{}),
				Eventtype:          "event",
				Reason:             "reason",
				Action:             "action",
				Note:               "note",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var msg EventMessage
			eventrecorder := &EventRecorder{
				EventfFn: func(m EventMessage) {
					msg = m
				},
			}
			tc.input(eventrecorder)
			if want, have := tc.expectedMsg, msg; want != have {
				t.Fatalf("unexpected msg: %v != %v", want, have)
			}
		})
	}
}

type podSmall struct{}

func (podSmall) GetName() string {
	return "good-pod"
}

func (podSmall) GetNamespace() string {
	return "test"
}

func (podSmall) GetUid() string {
	return "384900cd-dc7b-41ec-837e-9c4c1762363e"
}

func (podSmall) GetApiVersion() string {
	return ""
}

func (podSmall) GetKind() string {
	return "pod"
}

func (podSmall) GetResourceVersion() string {
	return "v1"
}
