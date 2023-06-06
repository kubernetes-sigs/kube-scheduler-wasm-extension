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
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// guestPool manages guest to pod assignments in a scheduling or binding cycle.
type guestPool[guest any] struct {
	// newGuest is a function to create a new guest.
	newGuest func(context.Context) (guest, error)

	// pool is a pool of unused guest instances.
	pool sync.Pool

	// schedulingPodUID is the UID of the pod that is being scheduled.
	// The plugin updates this field at the beginning of each scheduling cycle (PreFilter).
	//
	// Note: We cannot unassign the guest instance in PostFilter/Permit
	// because PostFilter is not enough to notice failures in the scheduling cycle,
	// it's called only when the Pod is rejected in PreFilter or Filter.
	// So, we need to trach Pods in the scheduling cycle and the binding cycle separately
	// so that we can know which guest instance to unassign at PreFilter.
	schedulingPodUID types.UID

	// assignedToSchedulingPod has the guest instance assignedToSchedulingPod to the pod.
	// assignedToSchedulingPod instance won't be put back to the pool until the scheduling cycle of this Pod is finished.
	// The plugin will update this field at the beginning of each scheduling cycle (PreFilter) along with schedulingPodUID.
	assignedToSchedulingPod guest

	// assignedToSchedulingPodLock is a lock to protect the access to the assigned instance.
	// The scheduler may call Filter(), AddPod(), RemovePod() concurrently, and we need to take a lock.
	// But, other methods of the plugin should be invoked for the same Pod serially.
	assignedToSchedulingPodLock sync.Mutex

	// assignedToBindingPod has the guest instances for Pods in binding cycle.
	assignedToBindingPod map[types.UID]guest
}

func newGuestPool[guest any](ctx context.Context, newGuest func(context.Context) (guest, error)) (*guestPool[guest], error) {
	p := &guestPool[guest]{
		newGuest:             newGuest,
		assignedToBindingPod: make(map[types.UID]guest),
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	g, createErr := newGuest(ctx)
	if createErr != nil {
		return nil, createErr
	}
	p.put(g)

	return p, nil
}

func (p *guestPool[guest]) getOrCreateGuest(ctx context.Context, podUID types.UID) (g guest, err error) {
	var ok bool
	if g, ok = p.getInstanceForSchedulingPod(ctx, podUID); !ok {
		if g, err = p.newGuest(ctx); err != nil {
			return
		}
	}

	// Assign the guest to the pod.
	p.assignForScheduling(g, podUID)

	return
}

// assignForScheduling assigns the guest instance to the pod in the scheduling
// cycle, so that the same pod can always get the same guest instance.
func (p *guestPool[guest]) assignForScheduling(g guest, podUID types.UID) {
	p.assignedToSchedulingPod = g
	p.schedulingPodUID = podUID
}

// assignForBinding assigns the guest instance to the pod in the binding cycle.
// This function doesn't take a lock because it is supposed to be called in the
// scheduling cycle: it will never be called in parallel.
func (p *guestPool[guest]) assignForBinding(podUID types.UID) {
	g := p.assignedToSchedulingPod
	p.assignedToBindingPod[podUID] = g
}

// unassignForScheduling unassigns the guest instance from the pod puts it back
// into the pool.
func (p *guestPool[guest]) unassignForScheduling() {
	assigned := p.assignedToSchedulingPod
	var zero guest
	p.assignedToSchedulingPod = zero
	p.schedulingPodUID = ""
	p.put(assigned)
}

// unassignForBinding unassigns the guest instance from the pod in the binding
// cycle.
func (p *guestPool[guest]) unassignForBinding(podUID types.UID) {
	assigned := p.assignedToBindingPod[podUID]
	delete(p.assignedToBindingPod, podUID)
	p.put(assigned)
}

// put puts the guest instance back to the pool.
func (p *guestPool[guest]) put(g guest) {
	p.pool.Put(g)
}

// getInstanceForSchedulingPod gets a guest instance for the Pod in the
// scheduling cycle. If the pod has already got a guest, it returns the guest.
// Otherwise, it gets a guest from the pool.
func (p *guestPool[guest]) getInstanceForSchedulingPod(ctx context.Context, podUID types.UID) (g guest, ok bool) {
	// Check if the pod has already got a guest.
	if podUID == p.schedulingPodUID {
		return p.assignedToSchedulingPod, true
	}

	// If not, get a guest from the pool.
	pG := p.pool.Get()
	if pG == nil {
		return
	}
	g, ok = pG.(guest)
	return
}
