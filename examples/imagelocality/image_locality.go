package main

import (
	"fmt"
	"strings"

	guestapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/handle/sharedlister/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/score"
)

// The two thresholds are used as bounds for the image score range. They correspond to a reasonable size range for
// container images compressed and stored in registries; 90%ile of images on dockerhub drops into this range.
const (
	mb                    int64 = 1024 * 1024
	minThreshold          int64 = 23 * mb
	maxContainerThreshold int64 = 1000 * mb
)

// imageLocality is a score plugin that favors nodes that already have requested pod container's images.
type imageLocality struct {
	sharedLister api.SharedLister
}

// Score invoked at the score extension point.
func (pl *imageLocality) Score(state guestapi.CycleState, pod proto.Pod, nodeName string) (int32, *guestapi.Status) {
	nodeInfo := pl.sharedLister.NodeInfos().Get(nodeName)
	if nodeInfo == nil {
		return 0, &guestapi.Status{Code: guestapi.StatusCodeError, Reason: fmt.Sprintf("failed to get node %q", nodeName)}
	}

	nodeInfos := pl.sharedLister.NodeInfos().List()
	if nodeInfos == nil {
		return 0, &guestapi.Status{Code: guestapi.StatusCodeError, Reason: "failed to list nodes"}
	}
	totalNumNodes := len(nodeInfos)

	imageScores := sumImageScores(nodeInfo, pod, totalNumNodes)
	score := calculatePriority(imageScores, len(pod.Spec().InitContainers)+len(pod.Spec().Containers))

	return int32(score), nil
}

// calculatePriority returns the priority of a node. Given the sumScores of requested images on the node, the node's
// priority is obtained by scaling the maximum priority value with a ratio proportional to the sumScores.
func calculatePriority(sumScores int64, numContainers int) int64 {
	maxThreshold := maxContainerThreshold * int64(numContainers)
	if sumScores < minThreshold {
		sumScores = minThreshold
	} else if sumScores > maxThreshold {
		sumScores = maxThreshold
	}

	return score.MaxNodeScore * (sumScores - minThreshold) / (maxThreshold - minThreshold)
}

// sumImageScores returns the sum of image scores of all the containers that are already on the node.
// Each image receives a raw score of its size, scaled by scaledImageScore. The raw scores are later used to calculate
// the final score.
func sumImageScores(nodeInfo guestapi.NodeInfo, pod proto.Pod, totalNumNodes int) int64 {
	var sum int64
	for _, container := range pod.Spec().InitContainers {
		if state, ok := nodeInfo.ImageStates()[normalizedImageName(*container.Image)]; ok {
			sum += scaledImageScore(state, totalNumNodes)
		}
	}
	for _, container := range pod.Spec().Containers {
		if state, ok := nodeInfo.ImageStates()[normalizedImageName(*container.Image)]; ok {
			sum += scaledImageScore(state, totalNumNodes)
		}
	}
	return sum
}

// scaledImageScore returns an adaptively scaled score for the given state of an image.
// The size of the image is used as the base score, scaled by a factor which considers how much nodes the image has "spread" to.
// This heuristic aims to mitigate the undesirable "node heating problem", i.e., pods get assigned to the same or
// a few nodes due to image locality.
func scaledImageScore(imageState *guestapi.ImageStateSummary, totalNumNodes int) int64 {
	spread := float64(imageState.NumNodes) / float64(totalNumNodes)
	return int64(float64(imageState.Size) * spread)
}

// normalizedImageName returns the CRI compliant name for a given image.
// TODO: cover the corner cases of missed matches, e.g,
// 1. Using Docker as runtime and docker.io/library/test:tag in pod spec, but only test:tag will present in node status
// 2. Using the implicit registry, i.e., test:tag or library/test:tag in pod spec but only docker.io/library/test:tag
// in node status; note that if users consistently use one registry format, this should not happen.
func normalizedImageName(name string) string {
	if strings.LastIndex(name, ":") <= strings.LastIndex(name, "/") {
		name = name + ":latest"
	}
	return name
}
