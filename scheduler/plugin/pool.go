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
//
// Assumptions made about the lifecycle are taken from the below diagram
// https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#extension-points
type guestPool[guest comparable] struct {
	// newGuest is a function to create a new guest.
	newGuest func(context.Context) (guest, error)

	mux sync.RWMutex

	// scheduledPodUID is the UID of the pod being scheduled.
	scheduledPodUID types.UID
	// scheduled is the guest being scheduled.
	scheduled guest

	// binding are any guests in the binding cycle.
	binding map[types.UID]guest

	// free pool of guests not in use
	free []guest
}

func newGuestPool[guest comparable](ctx context.Context, newGuest func(context.Context) (guest, error)) (*guestPool[guest], error) {
	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	g, createErr := newGuest(ctx)
	if createErr != nil {
		return nil, createErr
	}

	return &guestPool[guest]{
		newGuest: newGuest,
		binding:  make(map[types.UID]guest),
		free:     []guest{g},
	}, nil
}

// doWithSchedulingGuest runs the function with a guest used for scheduling
// cycles.
//
// There can be only one scheduling cycle in-progress and in most cases it is
// sequential. The only exception is preemption, a framework.PostFilterPlugin
// called when all nodes are filtered out, to make space available for the pod.
//
// The built-in defaultpreemption.DefaultPreemption might make parallel calls
// to wasmPlugin.Filter, wasmPlugin.AddPod and wasmPlugin.RemovePod in its
// `SelectVictimsOnNode` function.
//
// Hence, we need to serialize access to the scheduling guest, so that it isn't
// corrupted from overlapping use.
func (p *guestPool[guest]) doWithSchedulingGuest(ctx context.Context, podUID types.UID, fn func(guest)) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	// The scheduling cycle runs sequentially. If we still have an association,
	// take it over. Guests who cache state should use the podUID to identify a
	// difference.
	p.scheduledPodUID = podUID
	var zero guest
	if scheduled := p.scheduled; scheduled != zero {
		fn(scheduled)
		return nil
	}

	// Prefer the free pool
	if len(p.free) > 0 {
		g := p.free[0]
		p.free = p.free[1:]
		p.scheduled = g
		fn(g)
		return nil
	}

	// If we're at this point, the guest previously scheduled was re-assigned
	// to the binding cycle. Create a new guest.
	if g, err := p.newGuest(ctx); err == nil {
		p.scheduled = g
		fn(g)
		return nil
	} else {
		return err
	}
}

// getForBinding returns a guest for the current podUID or an error.
//
// There can be multiple scheduling cycles in-progress, but they always start
// after a schedule. If there's an existing association with the podUID, it
// is re-used. Otherwise, the current scheduling guest is re-associated for
// binding.
func (p *guestPool[guest]) getForBinding(podUID types.UID) guest {
	p.mux.Lock()
	defer p.mux.Unlock()

	// Fast path is we are in an existing binding cycle.
	var zero guest
	if g := p.binding[podUID]; g != zero {
		return g // current guest is still correct.
	}

	// We re-used the guest from the scheduling cycle for the binding cycle,
	// so that it doesn't have to unmarshal the pod again.
	if scheduled := p.scheduled; scheduled != zero {
		p.scheduledPodUID = ""
		p.scheduled = zero
		p.binding[podUID] = scheduled
		return scheduled
	}

	// Reaching here is unexpected, because the binding cycle must happen after
	// a scheduling one, even if binding cycles can run in parallel.
	panic("unexpected podUID")
}

// freeFromBinding should be called when a binding cycle ends for any reason.
func (p *guestPool[guest]) freeFromBinding(podUID types.UID) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if g, ok := p.binding[podUID]; ok {
		delete(p.binding, podUID)
		p.put(g)
	}
}

// put puts the guest instance back to the pool. This must be called under a
// lock.
func (p *guestPool[guest]) put(g guest) {
	var zero guest
	if g == zero {
		panic("nil guest")
	}
	p.free = append(p.free, g)
}
