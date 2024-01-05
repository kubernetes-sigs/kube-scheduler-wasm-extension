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

package proto

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

type object interface {
	GetMetadata() *meta.ObjectMeta
}

func GetName[O object](o O) string {
	if md := o.GetMetadata(); md != nil && md.Name != nil {
		return *md.Name
	}
	return ""
}

func GetNamespace[O object](o O) string {
	if md := o.GetMetadata(); md != nil && md.Namespace != nil {
		return *md.Namespace
	}
	return ""
}

func GetUid[O object](o O) string {
	if md := o.GetMetadata(); md != nil && md.Uid != nil {
		return *md.Uid
	}
	return ""
}

func GetResourceVersion[O object](o O) string {
	if md := o.GetMetadata(); md != nil && md.ResourceVersion != nil {
		return *md.ResourceVersion
	}
	return ""
}

var _ proto.Node = (*Node)(nil)

type Node struct {
	Msg *protoapi.Node
}

func (o *Node) GetName() string {
	return GetName(o.Msg)
}

func (o *Node) GetNamespace() string {
	return GetNamespace(o.Msg)
}

func (o *Node) GetUid() string {
	return GetUid(o.Msg)
}

func (o *Node) GetResourceVersion() string {
	return GetResourceVersion(o.Msg)
}

func (o *Node) GetKind() string {
	return "Node"
}

func (o *Node) GetApiVersion() string {
	return "v1"
}

func (o *Node) Spec() *protoapi.NodeSpec {
	return o.Msg.Spec
}

func (o *Node) Status() *protoapi.NodeStatus {
	return o.Msg.Status
}
