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
	"errors"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
)

func TestLog(t *testing.T) {
	tests := []struct {
		name             string
		severity         Severity
		input            func(api.Klog)
		expectedSeverity Severity
		expectedMsg      string
	}{
		{
			name: "nothing",
			input: func(klog api.Klog) {
				klog.Info()
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "\n",
		},
		{
			name: "adds newline",
			input: func(klog api.Klog) {
				klog.Info("hello world")
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "hello world\n",
		},
		{
			name:     "Error",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.Error("hello world")
			},
			expectedSeverity: ErrorLog,
			expectedMsg:      "hello world\n",
		},
		{
			name:     "Info: disabled",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.Info("hello world")
			},
		},
		{
			name:     "Error: disabled",
			severity: FatalLog,
			input: func(klog api.Klog) {
				klog.Error("hello world")
			},
		},
		{
			name: "no spaces between strings",
			input: func(klog api.Klog) {
				klog.Info("1", "2")
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "12\n",
		},
		{
			name: "spaces between non-strings",
			input: func(klog api.Klog) {
				klog.Info(1, 2)
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "1 2\n",
		},
		{
			name: "newline terminated",
			input: func(klog api.Klog) {
				klog.Info("hello", "\n")
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "hello\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				buf.Reset()
			}()

			var severity Severity
			var msg string
			klog := &Klog{
				Severity: tc.severity,
				LogFn: func(s Severity, m []byte) {
					severity = s
					msg = string(m)
				},
			}
			tc.input(klog)
			if want, have := tc.expectedSeverity, severity; want != have {
				t.Fatalf("unexpected severity: %v != %v", want, have)
			}
			if want, have := tc.expectedMsg, msg; want != have {
				t.Fatalf("unexpected msg: %v != %v", want, have)
			}
		})
	}
}

func TestLogS(t *testing.T) {
	tests := []struct {
		name             string
		severity         Severity
		input            func(api.Klog)
		expectedSeverity Severity
		expectedMsg      string
		expectedKVs      []string
	}{
		{
			name: "nothing",
			input: func(klog api.Klog) {
				klog.InfoS("")
			},
			expectedSeverity: InfoLog,
			expectedMsg:      "",
		},
		{
			name:     "ErrorS nil error",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.ErrorS(nil, "hello world")
			},
			expectedSeverity: ErrorLog,
			expectedMsg:      "hello world",
		},
		{
			name:     "ErrorS empty error",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.ErrorS(errors.New(""), "hello world")
			},
			expectedSeverity: ErrorLog,
			expectedMsg:      "hello world",
			expectedKVs:      []string{"err", ""},
		},
		{
			name:     "ErrorS empty msg",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.ErrorS(errors.New("error"), "")
			},
			expectedSeverity: ErrorLog,
			expectedKVs:      []string{"err", "error"},
		},
		{
			name:     "InfoS: disabled",
			severity: ErrorLog,
			input: func(klog api.Klog) {
				klog.InfoS("hello world")
			},
		},
		{
			name:     "ErrorS: disabled",
			severity: FatalLog,
			input: func(klog api.Klog) {
				klog.ErrorS(nil, "hello world")
			},
		},
		{
			name: "kvs: strings",
			input: func(klog api.Klog) {
				klog.InfoS("", "a", "1")
			},
			expectedSeverity: InfoLog,
			expectedKVs:      []string{"a", "1"},
		},
		{
			name: "kvs: duplicated", // host will decide what to do
			input: func(klog api.Klog) {
				klog.InfoS("", "a", "1", "a", "1")
			},
			expectedSeverity: InfoLog,
			expectedKVs:      []string{"a", "1", "a", "1"},
		},
		{
			name: "kvs: fmt.Stringer",
			input: func(klog api.Klog) {
				klog.InfoS("", "pod", api.KObj(podSmall{}))
			},
			expectedSeverity: InfoLog,
			expectedKVs:      []string{"pod", "test/good-pod"},
		},
		{
			name: "kvs: struct",
			input: func(klog api.Klog) {
				klog.InfoS("", "pod", podSmall{})
			},
			expectedSeverity: InfoLog,
			expectedKVs:      []string{"pod", "{}"},
		},
		{
			name: "kvs: redundant error",
			input: func(klog api.Klog) {
				klog.ErrorS(errors.New("ice"), "", "err", errors.New("cream"))
			},
			expectedSeverity: ErrorLog,
			expectedKVs:      []string{"err", "ice", "err", "cream"},
		},
		{
			name: "kvs: non-strings",
			input: func(klog api.Klog) {
				klog.InfoS("", 1, 2)
			},
			expectedSeverity: InfoLog,
			expectedKVs:      []string{"%!s(int=1)", "%!s(int=2)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				buf.Reset()
			}()

			var severity Severity
			var msg string
			var kvs []string
			klog := &Klog{
				Severity: tc.severity,
				LogSFn: func(s Severity, m string, kvBytes []byte) {
					severity = s
					msg = m
					if len(kvBytes) != 0 {
						// strip the last NUL character
						kvBytes = kvBytes[:len(kvBytes)-1]
						kvs = strings.Split(string(kvBytes), string('\x00'))
					}
				},
			}
			tc.input(klog)
			if want, have := tc.expectedSeverity, severity; want != have {
				t.Fatalf("unexpected severity: %v != %v", want, have)
			}
			if want, have := tc.expectedMsg, msg; want != have {
				t.Fatalf("unexpected msg: %v != %v", want, have)
			}
			if want, have := tc.expectedKVs, kvs; !reflect.DeepEqual(want, have) {
				t.Fatalf("unexpected kvs: %v != %v", want, have)
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

func (podSmall) GetResourceVersion() string {
	return "resource-version"
}
