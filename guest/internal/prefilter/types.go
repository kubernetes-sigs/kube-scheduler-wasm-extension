package prefilter

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	internalproto "sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/proto"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// CurrentPod is exposed for the cyclestate package.
// It is the current Pod being scheduled.
var CurrentPod proto.Pod = pod{}

// Nodes is exposed for the nodelister package.
var Nodes api.NodeInfoList = &nodeInfoList{}

// CycleState is exposed for the cyclestate package.
var CycleState api.CycleState = cycleState{}

var currentCycleState = map[string]any{}

type cycleState struct{}

func (cycleState) Read(key string) (val any, ok bool) {
	val, ok = currentCycleState[key]
	return
}

func (cycleState) Write(key string, val any) {
	currentCycleState[key] = val
}

func (cycleState) Delete(key string) {
	delete(currentCycleState, key)
}

type pod struct{}

func (pod) GetName() string {
	return internalproto.GetName(lazyPod())
}

func (pod) GetNamespace() string {
	return internalproto.GetNamespace(lazyPod())
}

func (pod) GetUid() string {
	return internalproto.GetUid(lazyPod())
}

func (pod) GetResourceVersion() string {
	return internalproto.GetResourceVersion(lazyPod())
}

func (pod) GetKind() string {
	return "Pod"
}

func (pod) GetApiVersion() string {
	return "v1"
}

func (pod) Spec() *protoapi.PodSpec {
	return lazyPod().Spec
}

func (pod) Status() *protoapi.PodStatus {
	return lazyPod().Status
}

var currentPod *protoapi.Pod

// lazyPod lazy initializes currentPod from imports.Pod.
func lazyPod() *protoapi.Pod {
	if pod := currentPod; pod != nil {
		return pod
	}

	var msg protoapi.Pod
	if err := imports.CurrentPod(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	currentPod = &msg
	return currentPod
}

var _ api.NodeInfoList = (*nodeInfoList)(nil)

type nodeInfoList struct{}

// currentNodeInfoList is a cache for a list of NodeInfo.
var currentNodeInfoList []api.NodeInfo

// isFullNodeInfoList indicates whether the cache is a full list or not.
var isFullNodeInfoList bool

func (n *nodeInfoList) Get(nodeName string) api.NodeInfo {
	for _, item := range currentNodeInfoList {
		// Try to find from the cache.
		if item.GetName() == nodeName {
			return item
		}
	}

	if isFullNodeInfoList {
		// If the cache is a full list, but we cannot find the node,
		// then this node is not found.
		return nil
	}

	// At this point, we don't fetch this Node, but just initialize nodeInfo.
	// When accessing the fields of nodeInfo, we will lazy-fetch the Node.
	ni := newNodeInfo(nodeName)

	// Store it into the cache so that we don't have to fetch it again.
	// In the same scheduling cycle, we always refer to a completely same Node data,
	// so caching nodes in the same scheduling cycle won't have an issue.
	currentNodeInfoList = append(currentNodeInfoList, ni)

	return ni
}

// List lists all NodeInfo of the cluster.
func (n *nodeInfoList) List() []api.NodeInfo {
	if isFullNodeInfoList {
		return currentNodeInfoList
	}

	var msg protoapi.NodeList
	if err := imports.Nodes(msg.UnmarshalVT); err != nil {
		panic(err)
	}

	size := len(msg.Items)
	if size == 0 {
		return nil
	}

	items := make([]api.NodeInfo, size)
	for i := range msg.Items {
		items[i] = &nodeInfo{
			node: &internalproto.Node{Msg: msg.Items[i]},
		}
	}
	currentNodeInfoList = items
	isFullNodeInfoList = true

	return items
}

var _ api.NodeInfo = (*nodeInfo)(nil)

type nodeInfo struct {
	name string

	node        proto.Node
	imageStates map[string]*api.ImageStateSummary
}

// newNodeInfo initializes a nodeInfo with the given nodeName.
// But, at this point, we don't fetch the Node data.
// They're fetched when actually accessing the fields of nodeInfo.
func newNodeInfo(nodeName string) *nodeInfo {
	return &nodeInfo{
		// Only nodename is the required to initiate nodeInfo.
		// Other fields will be lazy-fetched from the host, as necessary.
		name: nodeName,
	}
}

func (n *nodeInfo) GetUid() string {
	return n.lazyNode().GetUid()
}

func (n *nodeInfo) GetName() string {
	return n.name
}

func (n *nodeInfo) GetNamespace() string {
	return n.lazyNode().GetNamespace()
}

func (n *nodeInfo) GetResourceVersion() string {
	return n.lazyNode().GetResourceVersion()
}

func (n *nodeInfo) Node() proto.Node {
	return n.lazyNode()
}

// lazyNode actually fetches the node's data from the host.
func (n *nodeInfo) lazyNode() proto.Node {
	if n.node != nil {
		return n.node
	}

	var msg protoapi.Node
	if err := imports.Node(n.name, msg.UnmarshalVT); err != nil {
		panic(err)
	}

	n.node = &internalproto.Node{Msg: &msg}

	return n.node
}

func (n *nodeInfo) ImageStates() map[string]*api.ImageStateSummary {
	if n.imageStates != nil {
		return n.imageStates
	}

	// Fetch the image state summary from the host.
	n.imageStates = imports.NodeImageStates(n.name)

	return n.imageStates
}
