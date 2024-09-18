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

package wasm

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tetratelabs/wazero/experimental/wazerotest"
	v1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	k8stest "k8s.io/klog/v2/test"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

func Test_k8sKlogLogFn(t *testing.T) {
	var buf bytes.Buffer
	initKlog(t, &buf)

	// Configure the host to log info.
	h := host{logSeverity: severityInfo}

	// Create a fake wasm module, which has data the guest should write.
	mem := wazerotest.NewMemory(wazerotest.PageSize)
	mod := wazerotest.NewModule(mem)
	message := "hello"
	copy(mem.Bytes, message)

	// Invoke the host function in the same way the guest would have.
	h.k8sKlogLogFn(context.Background(), mod, []uint64{
		uint64(severityInfo), // severity
		0,                    // msg
		uint64(len(message)), // msg_len
	})

	want := message + "\n" // klog always adds newline
	if have := buf.String(); want != have {
		t.Fatalf("unexpected log message: %v != %v", want, have)
	}
}

func Test_k8sKlogLogsFn(t *testing.T) {
	var buf bytes.Buffer
	initKlog(t, &buf)

	// Configure the host to log info.
	h := host{logSeverity: severityInfo}

	// Create a fake wasm module, which has data the guest should write.
	mem := wazerotest.NewMemory(wazerotest.PageSize)
	mod := wazerotest.NewModule(mem)
	message := "hello"
	copy(mem.Bytes, message)
	kvs := "err\x00unhandled\u0000"
	copy(mem.Bytes[32:], kvs)

	// Invoke the host function in the same way the guest would have.
	h.k8sKlogLogsFn(context.Background(), mod, []uint64{
		uint64(severityInfo), // severity
		0,                    // msg
		uint64(len(message)), // msg_len
		32,                   // kvs
		uint64(len(kvs)),     // kvs_len
	})

	want := `"hello" err="unhandled"
` // klog always adds newline
	if have := buf.String(); want != have {
		t.Fatalf("unexpected log message: %v != %v", want, have)
	}
}

func initKlog(t *testing.T, buf *bytes.Buffer) {
	// Re-initialize klog for tests.
	fs := k8stest.InitKlog(t)
	// Disable timestamps.
	_ = fs.Set("skip_headers", "true")
	// Write log output to the buffer
	klog.SetOutput(buf)
}

type waitingPod struct {
	pod            *v1.Pod
	pendingPlugins map[string]*time.Timer
	s              chan *framework.Status
	mu             sync.RWMutex
}

func (wp *waitingPod) GetPod() *v1.Pod {
	return wp.pod
}

func (wp *waitingPod) GetPendingPlugins() []string {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	var plugins []string
	for plugin := range wp.pendingPlugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (wp *waitingPod) Allow(pluginName string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if timer, ok := wp.pendingPlugins[pluginName]; ok {
		timer.Stop()
		delete(wp.pendingPlugins, pluginName)
	}
}

func (wp *waitingPod) Reject(pluginName, msg string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if timer, ok := wp.pendingPlugins[pluginName]; ok {
		timer.Stop()
		delete(wp.pendingPlugins, pluginName)
	}
}

func Test_k8sHandleGetWaitingPodFn(t *testing.T) {
	recorder := &test.FakeRecorder{EventMsg: ""}
	uid := types.UID("c6feae3a-7082-42a5-a5ec-6ae2e1603727")

	// Create a fake WaitingPod
	pod := &v1.Pod{
		ObjectMeta: apimeta.ObjectMeta{
			Name:      "good-pod",
			Namespace: "test",
			UID:       uid,
		},
	}
	wp := &waitingPod{
		pod:            pod,
		pendingPlugins: make(map[string]*time.Timer),
		s:              make(chan *framework.Status, 1),
	}

	wp.mu.Lock()

	handle := &test.FakeHandle{
		Recorder:           recorder,
		GetWaitingPodValue: wp,
	}

	h := host{handle: handle}

	// Create a fake wasm module, which has data the guest should write.
	mem := wazerotest.NewMemory(wazerotest.PageSize)
	mod := wazerotest.NewModule(mem)
	copy(mem.Bytes, uid)

	// Invoke the host function in the same way the guest would have.
	h.k8sHandleGetWaitingPodFn(context.Background(), mod, []uint64{
		0,
		uint64(len(uid)),
		0,
		0,
	})

	// Checking the value returned by GetWaitingPod
	have := handle.GetWaitingPodValue.GetPod().GetUID()
	want := uid

	if want != have {
		t.Fatalf("unexpected uid: %v != %v", want, have)
	}
}

//func Test_k8sHandleEventRecorderEventFn(t *testing.T) {
//	recorder := &test.FakeRecorder{EventMsg: ""}
//	handle := &test.FakeHandle{Recorder: recorder}
//	h := host{handle: handle}
//
//	// Create a fake wasm module, which has data the guest should write.
//	mem := wazerotest.NewMemory(wazerotest.PageSize)
//	mod := wazerotest.NewModule(mem)
//	message := EventMessage{
//		RegardingReference: ObjectReference{},
//		RelatedReference:   ObjectReference{},
//		Eventtype:          "event",
//		Reason:             "reason",
//		Action:             "action",
//		Note:               "note",
//	}
//	jsonmsg, err := json.Marshal(message)
//	if err != nil {
//		t.Fatalf("error during json.Marshal %v", err)
//	}
//	copy(mem.Bytes, jsonmsg)
//
//	// Invoke the host function in the same way the guest would have.
//	h.k8sHandleEventRecorderEventfFn(context.Background(), mod, []uint64{
//		0,
//		uint64(len(jsonmsg)),
//	})
//
//	have := recorder.EventMsg
//	want := "event reason action note"
//
//	if want != have {
//		t.Fatalf("unexpected event: %v != %v", want, have)
//	}
//}

func Test_k8sHandleRejectWaitingPodFn(t *testing.T) {
	recorder := &test.FakeRecorder{EventMsg: ""}
	handle := &test.FakeHandle{Recorder: recorder}
	h := host{handle: handle}

	// Create a fake wasm module, which has data the guest should write.
	mem := wazerotest.NewMemory(wazerotest.PageSize)
	mod := wazerotest.NewModule(mem)
	uid := types.UID("c6feae3a-7082-42a5-a5ec-6ae2e1603727")
	copy(mem.Bytes, uid)

	// Invoke the host function in the same way the guest would have.
	h.k8sHandleRejectWaitingPodFn(context.Background(), mod, []uint64{
		0,
		uint64(len(uid)),
		0, // Ideally we should define some value, but we don't define it for now.
		0, // Ideally we should define some value, but we don't define it for now.
	})

	// Checking the value stored on handle
	have := handle.RejectWaitingPodValue
	want := uid

	if want != have {
		t.Fatalf("unexpected uid: %v != %v", want, have)
	}
}

//func Test_k8sHandleGetWaitingPodFn(t *testing.T) {
//	recorder := &test.FakeRecorder{EventMsg: ""}
//	handle := &test.FakeHandle{Recorder: recorder}
//	h := host{handle: handle}
//
//	// Create a fake wasm module, which has data the guest should write.
//	mem := wazerotest.NewMemory(wazerotest.PageSize)
//	mod := wazerotest.NewModule(mem)
//	uid := types.UID("c6feae3a-7082-42a5-a5ec-6ae2e1603727")
//	copy(mem.Bytes, uid)
//
//	// Invoke the host function in the same way the guest would have.
//	h.k8sHandleGetWaitingPodFn(context.Background(), mod, []uint64{
//		0,
//		uint64(len(uid)),
//		0,
//		0,
//	})
//
//	// Checking the value stored on handle
//	have := handle.GetWaitingPodValue
//	want := uid
//
//	if have == nil {
//		t.Fatalf("unexpected pod: %v != %v", want, have)
//	}
//}
