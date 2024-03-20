package test

import (
	"context"
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
}

func (h *FakeHandle) EventRecorder() events.EventRecorder {
	return h.Recorder
}

func (h *FakeHandle) AddNominatedPod(pod *framework.PodInfo, node *framework.NominatingInfo) {
}

func (h *FakeHandle) ClientSet() clientset.Interface {
	return nil
}

func (h *FakeHandle) DeleteNominatedPodIfExists(pod *v1.Pod) {
}

func (h *FakeHandle) Extenders() []framework.Extender {
	return nil
}

func (h *FakeHandle) KubeConfig() *restclient.Config {
	return nil
}

func (h *FakeHandle) SharedInformerFactory() informers.SharedInformerFactory {
	return nil
}

func (h *FakeHandle) RunFilterPluginsWithNominatedPods(ctx context.Context, state *framework.CycleState, pod *v1.Pod, info *framework.NodeInfo) (s *framework.Status) {
	return
}

func (h *FakeHandle) Parallelizer() (p parallelize.Parallelizer) {
	return
}

func (h *FakeHandle) GetWaitingPod(uid types.UID) (w framework.WaitingPod) {
	return
}

func (h *FakeHandle) IterateOverWaitingPods(callback func(framework.WaitingPod)) {
}

func (h *FakeHandle) NominatedPodsForNode(nodeName string) (f []*framework.PodInfo) {
	return
}

func (h *FakeHandle) RejectWaitingPod(uid types.UID) (b bool) {
	h.RejectWaitingPodValue = uid
	return uid == types.UID("handle-test")
}

func (h *FakeHandle) RunPreScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*v1.Node) (s *framework.Status) {
	return
}

func (h *FakeHandle) RunScorePlugins(context.Context, *framework.CycleState, *v1.Pod, []*v1.Node) (n []framework.NodePluginScores, s *framework.Status) {
	return
}

func (h *FakeHandle) RunFilterPlugins(context.Context, *framework.CycleState, *v1.Pod, *framework.NodeInfo) (s *framework.Status) {
	return
}

func (h *FakeHandle) RunPreFilterExtensionAddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *v1.Pod, nodeInfo *framework.NodeInfo) (s *framework.Status) {
	return
}

func (h *FakeHandle) RunPreFilterExtensionRemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *v1.Pod, nodeInfo *framework.NodeInfo) (s *framework.Status) {
	return
}

func (h *FakeHandle) SnapshotSharedLister() (s framework.SharedLister) {
	return
}

func (h *FakeHandle) UpdateNominatedPod(oldPod *v1.Pod, newPodInfo *framework.PodInfo) {
}
