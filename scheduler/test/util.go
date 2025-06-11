package test

import (
	"context"
	"errors"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/parallelize"
)

type FakeRecorder struct {
	EventMsg string
}

func (f *FakeRecorder) Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
	obj, ok := regarding.(*v1.ObjectReference)
	if !ok || obj.Name == "" {
		f.EventMsg = eventtype + " " + reason + " " + action + " " + note
	} else {
		f.EventMsg = obj.Name + " " + eventtype + " " + reason + " " + action + " " + note
	}
}

type FakeHandle struct {
	Recorder              events.EventRecorder
	RejectWaitingPodValue types.UID
	SharedLister          framework.SharedLister
	GetWaitingPodValue    framework.WaitingPod
}

func (h *FakeHandle) EventRecorder() events.EventRecorder {
	return h.Recorder
}

func (h *FakeHandle) AddNominatedPod(logger klog.Logger, pod *framework.PodInfo, node *framework.NominatingInfo) {
	panic("unimplemented")
}

func (h *FakeHandle) ClientSet() clientset.Interface {
	panic("unimplemented")
}

func (h *FakeHandle) DeleteNominatedPodIfExists(pod *v1.Pod) {
	panic("unimplemented")
}

func (h *FakeHandle) Extenders() []framework.Extender {
	panic("unimplemented")
}

func (h *FakeHandle) KubeConfig() *restclient.Config {
	panic("unimplemented")
}

func (h *FakeHandle) SharedInformerFactory() informers.SharedInformerFactory {
	panic("unimplemented")
}

func (h *FakeHandle) RunFilterPluginsWithNominatedPods(ctx context.Context, state *framework.CycleState, pod *v1.Pod, info *framework.NodeInfo) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) Parallelizer() (p parallelize.Parallelizer) {
	panic("unimplemented")
}

// waitingPod implements the framework.WaitingPod interface
type waitingPod struct {
	pod            *v1.Pod
	pendingPlugins map[string]*time.Timer
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
}

func (wp *waitingPod) Reject(reason string, msg string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
}

func NewWaitingPod(pod *v1.Pod, plugins map[string]*time.Timer) framework.WaitingPod {
	return &waitingPod{pod: pod, pendingPlugins: plugins}
}

// GetWaitingPod returns PodForHandleTest only when the uid is handle-test.
func (h *FakeHandle) GetWaitingPod(uid types.UID) framework.WaitingPod {
	if uid != types.UID("handle-test") {
		return nil
	}

	waitingPod := &waitingPod{
		pod:            PodForHandleTest,
		pendingPlugins: make(map[string]*time.Timer),
	}

	h.GetWaitingPodValue = waitingPod
	return waitingPod
}

func (h *FakeHandle) IterateOverWaitingPods(callback func(framework.WaitingPod)) {
	panic("unimplemented")
}

func (h *FakeHandle) NominatedPodsForNode(nodeName string) (f []*framework.PodInfo) {
	panic("unimplemented")
}

func (h *FakeHandle) RejectWaitingPod(uid types.UID) (b bool) {
	h.RejectWaitingPodValue = uid
	return uid == types.UID("handle-test")
}

func (h *FakeHandle) RunPreScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*framework.NodeInfo) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) RunScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*framework.NodeInfo) (n []framework.NodePluginScores, s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) RunFilterPlugins(context.Context, *framework.CycleState, *v1.Pod, *framework.NodeInfo) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) RunPreFilterExtensionAddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) RunPreFilterExtensionRemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) SnapshotSharedLister() framework.SharedLister {
	return h.SharedLister
}

func (h *FakeHandle) UpdateNominatedPod(logger klog.Logger, oldPod *v1.Pod, newPodInfo *framework.PodInfo) {
	panic("unimplemented")
}

func (h *FakeHandle) Activate(logger klog.Logger, pods map[string]*v1.Pod) {
	// No-op implementation for testing
}

func (h *FakeHandle) SharedDRAManager() framework.SharedDRAManager {
	panic("unimplemented")
}

type FakeSharedLister struct {
	NodeInfoLister framework.NodeInfoLister
}

func (c *FakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return c.NodeInfoLister
}

func (c *FakeSharedLister) StorageInfos() framework.StorageInfoLister {
	panic("unimplemented")
}

type FakeNodeInfoLister struct {
	Nodes []*framework.NodeInfo
}

func (c *FakeNodeInfoLister) List() ([]*framework.NodeInfo, error) {
	return c.Nodes, nil
}

func (c *FakeNodeInfoLister) HavePodsWithAffinityList() ([]*framework.NodeInfo, error) {
	panic("unimplemented")
}

func (c *FakeNodeInfoLister) HavePodsWithRequiredAntiAffinityList() ([]*framework.NodeInfo, error) {
	panic("unimplemented")
}

func (c *FakeNodeInfoLister) Get(name string) (*framework.NodeInfo, error) {
	for _, n := range c.Nodes {
		if n.Node().Name == name {
			return n, nil
		}
	}

	return nil, errors.New("not found")
}
