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
	"fmt"
	"strings"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
)

// Note: This includes some functions in k8s.io/klog/v2 k8s_references.go,
// customized from the original source where noted.

// logBuf is shared as logging cannot happen concurrently in wasm. By sharing
// a buffer, we reduce log allocation/GC overhead.
var logBuf strings.Builder

// writeRef writes the reference to the shared log buffer
func writeRef(ns, name string) {
	logBuf.Grow(len(ns) + len(name) + 1)
	logBuf.WriteString(ns)
	logBuf.WriteRune('/')
	logBuf.WriteString(name)
}

type objectRef struct {
	proto.Metadata
}

// String implements fmt.Stringer
func (ref *objectRef) String() string {
	obj := ref.Metadata
	if obj == nil {
		return ""
	}
	if name, ns := obj.GetName(), obj.GetNamespace(); ns != "" {
		logBuf.Reset()
		writeRef(ns, name)
		return logBuf.String()
	} else {
		return name
	}
}

// KObj wraps a proto.Metadata to normalize log values.
//
// Note: This is similar klog.KObj, except references proto.Metadata and avoids
// reflection calls that don't compile in TinyGo 0.28
func KObj(obj proto.Metadata) fmt.Stringer {
	return &objectRef{obj}
}

// KObjSlice wraps a slice of proto.Metadata to normalize log values.
//
// Note: This is similar klog.KObjSlice, except references proto.Metadata and
// avoids reflection calls that don't compile in TinyGo 0.28
func KObjSlice[M proto.Metadata](objs []M) fmt.Stringer {
	return &kobjSlice[M]{objs}
}

// kobjSlice is a normalized logging reference of proto.Metadata. Construct
// this using KObjSlice.
//
// Note: This is like klog.KObjSlice except lazy to avoid eagerly unmarshalling
// protos.
type kobjSlice[M proto.Metadata] struct {
	objs []M
}

// String implements fmt.Stringer
func (ks *kobjSlice[M]) String() string {
	return kobjSliceString(ks.objs)
}

// KObjSliceFn wraps function that produces a slice of proto.Metadata to
// normalize log values.
//
// Note: This is the same as KObjSlice, except avoids calling a function when
// logging is disabled.
func KObjSliceFn[M proto.Metadata](lazy func() []M) fmt.Stringer {
	return &kObjSliceFn[M]{lazy}
}

type kObjSliceFn[M proto.Metadata] struct {
	fn func() []M
}

// String implements fmt.Stringer
func (kl *kObjSliceFn[M]) String() string {
	if fn := kl.fn; fn == nil {
		return "[]"
	} else {
		return kobjSliceString(fn())
	}
}

func kobjSliceString[M proto.Metadata](objs []M) string {
	if len(objs) == 0 {
		return "[]"
	}
	logBuf.Reset()
	logBuf.WriteRune('[')
	for i := range objs {
		if i > 0 {
			logBuf.WriteRune(' ')
		}
		obj := objs[i]
		if name, ns := obj.GetName(), obj.GetNamespace(); ns != "" {
			writeRef(ns, name)
		} else {
			logBuf.WriteString(name)
		}
	}
	logBuf.WriteRune(']')
	return logBuf.String()
}
