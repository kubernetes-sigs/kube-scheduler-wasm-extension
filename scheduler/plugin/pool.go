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
)

// guestPool manages guest to pod assignments in a scheduling or binding cycle.
//
// Assumptions made about the lifecycle are taken from the below diagram
// https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#extension-points
type guestPool[guest comparable] struct {
	// newGuest is a function to create a new guest.
	newGuest func(context.Context) (guest, error)

	mux sync.RWMutex

	// scheduled is up to one guest in the scheduling cycle.
	schedulingCycleID uint32
	scheduled         guest

	// binding are any guests in the binding cycle.
	binding map[uint32]guest

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
		binding:  make(map[uint32]guest),
		free:     []guest{g},
	}, nil
}

// getForScheduling returns a guest for the current cycleID or an error.
//
// There can be up to one scheduling cycle in-progress, so this returns the
// current guest, possibly reclaimed from a prior cycle. Otherwise, one will
// be created.
func (p *guestPool[guest]) getForScheduling(ctx context.Context, cycleID uint32) (guest, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	// The scheduling cycle runs sequentially. If we still have an association,
	// take it over.
	p.schedulingCycleID = cycleID

	var zero guest
	if scheduled := p.scheduled; scheduled != zero {
		// TODO: consider explicitly resetting the guest when
		// schedulingCycleID != cycleID
		return scheduled, nil
	}

	// Prefer the free pool
	if len(p.free) > 0 {
		g := p.free[0]
		p.free = p.free[1:]
		p.scheduled = g
		return g, nil
	}

	// If we're at this point, the guest previously scheduled was re-assigned
	// to the binding cycle. Create a new guest.
	if g, err := p.newGuest(ctx); err == nil {
		p.scheduled = g
		return g, nil
	} else {
		return zero, err
	}
}

// getForBinding returns a guest for the current cycleID or an error.
//
// There can be multiple scheduling cycles in-progress, but they always start
// after a schedule. If there's an existing association with the cycleID, it
// is re-used. Otherwise, the current scheduling guest is re-associated for
// binding.
func (p *guestPool[guest]) getForBinding(cycleID uint32) guest {
	p.mux.Lock()
	defer p.mux.Unlock()

	// Fast path is we are in an existing binding cycle.
	var zero guest
	if g := p.binding[cycleID]; g != zero {
		return g // current guest is still correct.
	}

	// The pod pointer of the scheduling cycle will differ from the pointer in
	// the binding one. Take over the currently scheduled guest.
	if scheduled := p.scheduled; scheduled != zero {
		p.schedulingCycleID = 0
		p.scheduled = zero
		p.binding[cycleID] = scheduled
		return scheduled
	}

	// Reaching here is unexpected, because the binding cycle must happen after
	// a scheduling one, even if binding cycles can run in parallel.
	panic("unexpected pod pointer")
}

// freeFromBinding should be called when a binding cycle ends for any reason.
func (p *guestPool[guest]) freeFromBinding(cycleID uint32) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if g, ok := p.binding[cycleID]; ok {
		delete(p.binding, cycleID)
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

	// TODO consider allowing the guest to reset its state

	p.free = append(p.free, g)
}
