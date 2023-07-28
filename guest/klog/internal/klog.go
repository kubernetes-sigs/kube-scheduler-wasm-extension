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

// Package internal allows unit testing without requiring wasm imports.
package internal

import (
	"bytes"
	"fmt"
	"strings"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
)

// Severity is the same as severity.Severity in klog
type Severity = int32

const (
	InfoLog Severity = iota
	WarningLog
	ErrorLog
	FatalLog
)

type Klog struct {
	api.UnimplementedKlog

	Severity
	LogFn  func(severity Severity, msg []byte)
	LogSFn func(severity Severity, msg string, kvs []byte)
}

// Info implements the same method as documented on api.Klog.
func (k *Klog) Info(args ...any) {
	k.log(InfoLog, args)
}

// Error implements the same method as documented on api.Klog.
func (k *Klog) Error(args ...any) {
	k.log(ErrorLog, args)
}

// InfoS implements the same method as documented on api.Klog.
func (k *Klog) InfoS(msg string, keysAndValues ...any) {
	k.logs(InfoLog, msg, keysAndValues)
}

// ErrorS implements the same method as documented on api.Klog.
func (k *Klog) ErrorS(err error, msg string, keysAndValues ...any) {
	if err != nil {
		errKV := [2]any{"err", err}
		keysAndValues = append(errKV[:], keysAndValues...)
	}
	k.logs(ErrorLog, msg, keysAndValues)
}

// log coerces the args to a string and logs them, when the severity is loggable.
func (k *Klog) log(severity Severity, args []any) {
	if severity < k.Severity {
		return // don't incur host call overhead
	}
	msg := logString(args)
	k.LogFn(severity, msg)
}

// logs encodes `keysAndValues` as a NUL-terminated string, when the severity
// is loggable.
func (k *Klog) logs(severity Severity, msg string, keysAndValues []any) {
	if severity < k.Severity {
		return // don't incur host call overhead
	}
	k.LogSFn(severity, msg, logKVs(keysAndValues))
}

// buf is a reusable unbounded buffer.
var buf bytes.Buffer

// logString returns the bytes representing the args joined like Klog.Info.
func logString(args []any) []byte {
	buf.Reset()
	_, _ = fmt.Fprint(&buf, args...)
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// logKVs makes a NUL-terminated string of values
func logKVs(kvs []any) []byte {
	buf.Reset()

	count := len(kvs)
	if count == 0 {
		return nil
	}

	for i := 0; i < count; i++ {
		var s string
		if i%2 == 0 { // key
			switch k := kvs[i].(type) {
			case string:
				s = k
			default:
				s = fmt.Sprintf("%s", k)
			}
		} else { // value
			switch v := kvs[i].(type) {
			case fmt.Stringer:
				s = v.String()
			case string:
				s = v
			case error:
				s = v.Error()
			default:
				s = fmt.Sprintf("%s", v)
			}
		}
		if strings.ContainsRune(s, '\x00') {
			panic(fmt.Errorf("invalid log message %q", s))
		}
		buf.WriteString(s)
		buf.WriteByte(0) // NUL-terminator
	}
	return buf.Bytes()
}
