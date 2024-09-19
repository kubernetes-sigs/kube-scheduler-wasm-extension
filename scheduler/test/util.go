package test

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/parallelize"
)

type FakeRecorder struct {
	EventMsg string
}

func (f *FakeRecorder) Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
	obj, ok := regarding.(*v1.ObjectReference)
	if !ok || obj.Name == "" {
		f.EventMsg = fmt.Sprintf(eventtype + " " + reason + " " + action + " " + note)
	} else {
		f.EventMsg = fmt.Sprintf(obj.Name + " " + eventtype + " " + reason + " " + action + " " + note)
	}
}

type FakeHandle struct {
	Recorder              events.EventRecorder
	RejectWaitingPodValue types.UID
	SharedLister          framework.SharedLister
}

func (h *FakeHandle) EventRecorder() events.EventRecorder {
	return h.Recorder
}

func (h *FakeHandle) AddNominatedPod(pod *framework.PodInfo, node *framework.NominatingInfo) {
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

func (h *FakeHandle) GetWaitingPod(uid types.UID) (w framework.WaitingPod) {
	panic("unimplemented")
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

func (h *FakeHandle) RunPreScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*v1.Node) (s *framework.Status) {
	panic("unimplemented")
}

func (h *FakeHandle) RunScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*v1.Node) (n []framework.NodePluginScores, s *framework.Status) {
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

func (h *FakeHandle) UpdateNominatedPod(oldPod *v1.Pod, newPodInfo *framework.PodInfo) {
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
