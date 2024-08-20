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
	"context"
	"encoding/json"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	i32                                   = wazeroapi.ValueTypeI32
	i64                                   = wazeroapi.ValueTypeI64
	k8sApi                                = "k8s.io/api"
	k8sApiNode                            = "node"
	k8sApiNodeList                        = "nodeList"
	k8sApiNodeToStatusMap                 = "nodeToStatusMap"
	k8sKlog                               = "k8s.io/klog"
	k8sKlogLog                            = "log"
	k8sKlogLogs                           = "logs"
	k8sKlogSeverity                       = "severity"
	k8sScheduler                          = "k8s.io/scheduler"
	k8sSchedulerCurrentNodeName           = "currentNodeName"
	k8sSchedulerTargetPod                 = "targetPod"
	k8sSchedulerFilteredNodeList          = "filteredNodeList"
	k8sSchedulerCurrentPod                = "currentPod"
	k8sSchedulerGetConfig                 = "get_config"
	k8sSchedulerNodeScoreList             = "nodeScoreList"
	k8sSchedulerNodeImageStates           = "nodeImageStates"
	k8sSchedulerResultClusterEvents       = "result.cluster_events"
	k8sSchedulerResultNodeNames           = "result.node_names"
	k8sSchedulerResultNominatedNodeName   = "result.nominated_node_name"
	k8sSchedulerResultStatusReason        = "result.status_reason"
	k8sSchedulerResultNormalizedScoreList = "result.normalized_score_list"
	k8sSchedulerHandleEventRecorderEventf = "handle.eventrecorder.eventf"
	k8sSchedulerHandleRejectWaitingPod    = "handle.reject_waiting_pod"
)

func instantiateHostApi(ctx context.Context, runtime wazero.Runtime, handle framework.Handle) (wazeroapi.Module, error) {
	host := &host{handle: handle}
	return runtime.NewHostModuleBuilder(k8sApi).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sApiNodeFn), []wazeroapi.ValueType{i32, i32, i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("nodename", "nodename_len", "buf", "buf_limit").Export(k8sApiNode).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sApiNodeListFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiNodeList).
		Instantiate(ctx)
}

func instantiateHostKlog(ctx context.Context, runtime wazero.Runtime, logSeverity int32) (wazeroapi.Module, error) {
	host := &host{logSeverity: logSeverity}
	return runtime.NewHostModuleBuilder(k8sKlog).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sKlogLogFn), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("severity", "msg", "msg_len").Export(k8sKlogLog).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sKlogLogsFn), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("severity", "msg", "msg_len", "kvs", "kvs_len").Export(k8sKlogLogs).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sKlogSeverityFn), []wazeroapi.ValueType{}, []wazeroapi.ValueType{i32}).
		WithResultNames("severity").Export(k8sKlogSeverity).
		Instantiate(ctx)
}

func instantiateHostScheduler(ctx context.Context, runtime wazero.Runtime, guestConfig string, handle framework.Handle) (wazeroapi.Module, error) {
	host := &host{guestConfig: guestConfig, handle: handle}
	return runtime.NewHostModuleBuilder(k8sScheduler).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sSchedulerGetConfigFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sSchedulerGetConfig).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerApiFilteredNodeListFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sSchedulerFilteredNodeList).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerTargetPodFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sSchedulerTargetPod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerCurrentNodeNameFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sSchedulerCurrentNodeName).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sSchedulerNodeImageStatesFn), []wazeroapi.ValueType{i32, i32, i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("nodename", "nodename_len", "buf", "buf_limit").Export(k8sSchedulerNodeImageStates).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerCurrentPodFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sSchedulerCurrentPod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultClusterEventsFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultClusterEvents).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultNodeNamesFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultNodeNames).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultNominatedNodeNameFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultNominatedNodeName).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultStatusReasonFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultStatusReason).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerNodeToStatusMapFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(k8sApiNodeToStatusMap).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerResultNormalizedScoreListFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerResultNormalizedScoreList).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(k8sSchedulerNodeScoreListFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerNodeScoreList).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sHandleEventRecorderEventfFn), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerHandleEventRecorderEventf).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(host.k8sHandleRejectWaitingPodFn), []wazeroapi.ValueType{i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(k8sSchedulerHandleRejectWaitingPod).
		Instantiate(ctx)
}

// stackKey is a context.Context value associated with a stack
// pointer to the current request.
type stackKey struct{}

// stack holds any parameters or results from functions implemented by the
// guest. An instance of stack is only used for a single function invocation,
// such as guest.filterFn.
//
// # Notes
//
//   - This is needed because WebAssembly types are numeric only.
//   - Result fields are conventionally prefixed with "result".
//   - Declaring one type is less complicated than one+context key per
//     function. Functions should ignore fields they don't use.
type stack struct {
	// filteredNodes are used by guest.prescoreFn
	filteredNodes []*v1.Node

	// currentNodeName is a Node's name that is being evaluated.
	currentNodeName string

	// currentPod is used by guest.filterFn and guest.scoreFn
	currentPod *v1.Pod

	// nodeToStatusMap is used by guest.postfilterFn
	nodeToStatusMap map[string]*framework.Status

	// nodeScoreList is used by guest.normalizedscoreFn
	nodeScoreList framework.NodeScoreList

	// resultClusterEvents is returned by guest.enqueueFn
	resultClusterEvents []framework.ClusterEvent

	// resultNodeNames is returned by guest.prefilterFn
	resultNodeNames []string

	// resultNominatedNodeName is returned by guest.postfilterFn
	resultNominatedNodeName string

	// reason returned by all guest exports except guest.enqueueFn
	//
	// It is a field to avoid compiler-specific malloc/free functions, and to
	// avoid having to deal with out-params because TinyGo only supports a
	// single result.
	resultStatusReason string

	// resultNormalizedScoreList is returned by guest.normalizedscoreFn
	resultNormalizedScoreList framework.NodeScoreList

	// targetPod is the target Pod for this operation,
	// which is supposed to be used by AddPod/RemovePod in PreFilterExtension.
	targetPod *v1.Pod
}

func paramsFromContext(ctx context.Context) *stack {
	return ctx.Value(stackKey{}).(*stack)
}

func (h host) k8sApiNodeFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	nodename := uint32(stack[0])
	nodenameLen := uint32(stack[1])
	buf := uint32(stack[2])
	bufLimit := bufLimit(stack[3])

	var nodeName string
	if b, ok := mod.Memory().Read(nodename, nodenameLen); !ok {
		panic("out of memory reading nodeName")
	} else {
		nodeName = string(b)
	}

	var node *v1.Node
	nodeinfo, err := h.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err == nil {
		node = nodeinfo.Node()
	}

	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), node, buf, bufLimit))
}

func (h host) k8sApiNodeListFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeinfos, err := h.handle.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		panic(err)
	}

	nodes := make([]v1.Node, 0, len(nodeinfos))
	for _, ni := range nodeinfos {
		nodes = append(nodes, *ni.Node())
	}

	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), &v1.NodeList{Items: nodes}, buf, bufLimit))
}

func k8sSchedulerApiFilteredNodeListFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodes := paramsFromContext(ctx).filteredNodes
	// Use v1.NodeList to encode the nodes, as it is easier for both sides.
	nl := make([]string, len(nodes))
	for i := range nodes {
		nl[i] = nodes[i].GetName()
	}

	b, err := json.Marshal(nl)
	if err != nil {
		panic(err)
	}
	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), string(b), buf, bufLimit))
}

func k8sSchedulerCurrentNodeNameFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeName := paramsFromContext(ctx).currentNodeName

	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), nodeName, buf, bufLimit))
}

func k8sSchedulerCurrentPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	podInfo := paramsFromContext(ctx).currentPod
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), podInfo, buf, bufLimit))
}

// k8sSchedulerTargetPodFn is a function used by the host to send the podInfo.
func k8sSchedulerTargetPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	podInfo := paramsFromContext(ctx).targetPod
	stack[0] = uint64(marshalIfUnderLimit(mod.Memory(), podInfo, buf, bufLimit))
}

// k8sSchedulerNodeToStatusMapFn is a function used by the host to send the nodeStatusMap.
func k8sSchedulerNodeToStatusMapFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeToStatusMap := paramsFromContext(ctx).nodeToStatusMap
	nodeCodeMap := nodeStatusMapToMap(nodeToStatusMap)
	mapByte, err := json.Marshal(nodeCodeMap)
	if err != nil {
		panic(err)
	}
	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), string(mapByte), buf, bufLimit))
}

type host struct {
	guestConfig string
	logSeverity int32
	handle      framework.Handle
}

func (h host) k8sSchedulerGetConfigFn(_ context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	config := h.guestConfig

	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), config, buf, bufLimit))
}

func (h host) k8sSchedulerNodeImageStatesFn(_ context.Context, mod wazeroapi.Module, stack []uint64) {
	nodename := uint32(stack[0])
	nodenameLen := uint32(stack[1])
	buf := uint32(stack[2])
	bufLimit := bufLimit(stack[3])

	var nodeName string
	if b, ok := mod.Memory().Read(nodename, nodenameLen); !ok {
		panic("out of memory reading nodeName")
	} else {
		nodeName = string(b)
	}

	var imageStates map[string]*framework.ImageStateSummary
	ni, err := h.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err == nil {
		imageStates = ni.ImageStates
	}

	b, err := json.Marshal(imageStates)
	if err != nil {
		panic(err)
	}
	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), string(b), buf, bufLimit))
}

const (
	severityInfo int32 = iota
	severityWarning
	severityError
	severityFatal
)

// k8sKlogLogFn is a function used by the wasm guest to access klog.Info and
// klog.Error.
func (h host) k8sKlogLogFn(_ context.Context, mod wazeroapi.Module, stack []uint64) {
	severity := int32(stack[0])
	msg := uint32(stack[1])
	msgLen := uint32(stack[2])

	if severity > h.logSeverity {
		return
	}

	if b, ok := mod.Memory().Read(msg, msgLen); !ok {
		// don't panic if we can't read the message.
	} else {
		switch severity {
		case severityInfo:
			klog.Info(string(b))
		case severityError:
			klog.Error(string(b))
		}
	}
}

// k8sKlogLogsFn is a function used by the wasm guest to access klog.InfoS and
// klog.ErrorS.
func (h host) k8sKlogLogsFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	severity := int32(stack[0])
	msg := uint32(stack[1])
	msgLen := uint32(stack[2])
	kvs := uint32(stack[3])
	kvsLen := uint32(stack[4])

	// no key-values is unlikely, but possible
	if kvsLen == 0 {
		h.k8sKlogLogFn(ctx, mod, stack)
		return
	}

	if severity < h.logSeverity {
		return
	}

	var msgS string
	if b, ok := mod.Memory().Read(msg, msgLen); !ok {
		return // don't panic if we can't read the message.
	} else {
		msgS = string(b)
	}

	var kvsS []any
	if b, ok := mod.Memory().Read(kvs, kvsLen); !ok {
		return // don't panic if we can't read the kvs.
	} else if strings := fromNULTerminated(b); len(strings) > 0 {
		kvsS = make([]any, len(strings))
		for i := range strings {
			kvsS[i] = strings[i]
		}
	}

	switch severity {
	case severityInfo:
		klog.InfoS(msgS, kvsS...)
	case severityError:
		klog.ErrorS(nil, msgS, kvsS...)
	}
}

// k8sKlogSeverityFn is a function used by the wasm guest to obviate log
// overhead when a message won't be written.
func (h host) k8sKlogSeverityFn(_ context.Context, _ wazeroapi.Module, stack []uint64) {
	stack[0] = uint64(h.logSeverity)
}

// k8sSchedulerResultClusterEventsFn is a function used by the wasm guest to set the
// cluster events result from guestExportEnqueue.
func k8sSchedulerResultClusterEventsFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var clusterEvents []framework.ClusterEvent
	if b, ok := mod.Memory().Read(buf, bufLen); !ok {
		panic("out of memory reading clusterEvents")
	} else {
		clusterEvents = decodeClusterEvents(b)
	}
	paramsFromContext(ctx).resultClusterEvents = clusterEvents
}

// k8sSchedulerResultNodeNamesFn is a function used by the wasm guest to set the
// node names result from guestExportPreFilter.
func k8sSchedulerResultNodeNamesFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var nodeNames []string
	if b, ok := mod.Memory().Read(buf, bufLen); !ok {
		panic("out of memory reading nodeNames")
	} else {
		nodeNames = fromNULTerminated(b)
	}
	paramsFromContext(ctx).resultNodeNames = nodeNames
}

// k8sSchedulerResultNominatedNodeNameFn is a function used by the wasm guest to set the
// nominated node name result from guestExportPostFilter.
func k8sSchedulerResultNominatedNodeNameFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var nominatedNodeName string
	if b, ok := mod.Memory().Read(buf, bufLen); !ok {
		panic("out of memory reading nominatedNodeName")
	} else {
		nominatedNodeName = string(b)
	}
	paramsFromContext(ctx).resultNominatedNodeName = nominatedNodeName
}

// k8sSchedulerResultStatusReasonFn is a function used by the wasm guest to set the
// framework.Status reason result from all functions.
func k8sSchedulerResultStatusReasonFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var reason string
	if b, ok := mod.Memory().Read(buf, bufLen); !ok {
		// don't panic if we can't read the message.
		reason = "BUG: out of memory reading message"
	} else {
		reason = string(b)
	}
	paramsFromContext(ctx).resultStatusReason = reason
}

// Converts nodeToStatusMap to a map with node names as keys and their scores as integer values.
func nodeStatusMapToMap(originalMap map[string]*framework.Status) map[string]int {
	newMap := make(map[string]int)
	for key, value := range originalMap {
		if value != nil {
			newMap[key] = int(value.Code())
		}
	}
	return newMap
}

// k8sSchedulerNodeScoreListFn is a function used by the host to send the nodeScoreList.
func k8sSchedulerNodeScoreListFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	nodeScoreList := paramsFromContext(ctx).nodeScoreList
	nodeCodeMap := NodeScoreListToMap(nodeScoreList)
	mapByte, err := json.Marshal(nodeCodeMap)
	if err != nil {
		panic(err)
	}
	stack[0] = uint64(writeStringIfUnderLimit(mod.Memory(), string(mapByte), buf, bufLimit))
}

// k8sSchedulerResultNormalizedScoreListFn is a function used by the wasm guest to set the
// nodeScoreList result from guestExportNormalizeScore.
func k8sSchedulerResultNormalizedScoreListFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var nodeScoreList map[string]int
	b, ok := mod.Memory().Read(buf, bufLen)
	if !ok {
		panic("out of memory reading normalized score list")
	}
	if err := json.Unmarshal(b, &nodeScoreList); err != nil {
		panic(err)
	}
	paramsFromContext(ctx).resultNormalizedScoreList = MapToNodeScoreList(nodeScoreList)
}

// Converts a list of framework.NodeScore to a map with node names as keys and their scores as integer values.
func NodeScoreListToMap(nodeScoreList []framework.NodeScore) map[string]int {
	scoreMap := make(map[string]int)
	for _, nodeScore := range nodeScoreList {
		scoreMap[nodeScore.Name] = int(nodeScore.Score)
	}
	return scoreMap
}

// Transforms a map of node names and scores (as integers) into a slice of framework.NodeScore structures.
func MapToNodeScoreList(scoreMap map[string]int) []framework.NodeScore {
	var nodeScoreList []framework.NodeScore
	for nodeName, score := range scoreMap {
		nodeScoreList = append(nodeScoreList, framework.NodeScore{
			Name:  nodeName,
			Score: int64(score),
		})
	}
	return nodeScoreList
}

// k8sHandleEventRecorderEventfFn is a function used by the wasm guest to call EventRecorder.Eventf
func (h host) k8sHandleEventRecorderEventfFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLen := uint32(stack[1])

	var msg EventMessage
	b, ok := mod.Memory().Read(buf, bufLen)
	if !ok {
		panic("out of memory reading eventrecorder event")
	}
	if err := json.Unmarshal(b, &msg); err != nil {
		panic(err)
	}
	regardingObj := convertToObjectReference(&msg.RegardingReference)
	relatedObj := convertToObjectReference(&msg.RelatedReference)
	evt := h.handle.EventRecorder()
	evt.Eventf(regardingObj, relatedObj, msg.Eventtype, msg.Reason, msg.Action, msg.Note, nil)
}

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

func convertToObjectReference(objRef *ObjectReference) *v1.ObjectReference {
	return &v1.ObjectReference{
		Kind:            objRef.Kind,
		APIVersion:      objRef.APIVersion,
		Name:            objRef.Name,
		Namespace:       objRef.Namespace,
		UID:             types.UID(objRef.UID),
		ResourceVersion: objRef.ResourceVersion,
	}
}

// k8sHandleRejectWaitingPodFn is a function used by the wasm guest to call RejectWaitingPod
func (h host) k8sHandleRejectWaitingPodFn(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	iBuf := uint32(stack[0])
	iBufLen := uint32(stack[1])
	oBuf := uint32(stack[2])
	oBufLimit := uint32(stack[3])

	b, ok := mod.Memory().Read(iBuf, iBufLen)
	if !ok {
		panic("out of memory reading rejectWaitingPod")
	}
	uid := types.UID(b)
	IsRejected := h.handle.RejectWaitingPod(uid)
	wasmBool := uint64(0)
	if IsRejected {
		wasmBool = uint64(1)
	}
	writeUint64(mod.Memory(), wasmBool, oBuf, oBufLimit)
}
