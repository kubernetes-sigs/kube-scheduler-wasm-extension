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
	"reflect"
	"testing"
)

var ctx = context.Background()

type testGuest struct {
	val int
}

func Test_guestPool_getForScheduling(t *testing.T) {
	id := uint32(1)
	differentID := uint32(2)

	var counter int
	pl, err := newGuestPool(ctx, func(ctx2 context.Context) (*testGuest, error) {
		counter++
		return &testGuest{val: counter}, nil
	})
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	g1, err := pl.getForScheduling(ctx, id)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g1 == nil {
		t.Fatalf("have nil guest instance")
	}

	// Scheduling is sequential, so we expect a different ID to re-use the prior
	g2, err := pl.getForScheduling(ctx, differentID)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g2 == nil {
		t.Fatalf("have nil guest instance")
	}
	if want, have := g1, g2; !reflect.DeepEqual(want, have) {
		t.Fatalf("expected the same guest: want %v, have %v", want, have)
	}
}

func Test_guestPool_getForBinding(t *testing.T) {
	id := uint32(1)
	differentID := uint32(2)

	var counter int
	pl, err := newGuestPool(ctx, func(ctx2 context.Context) (*testGuest, error) {
		counter++
		return &testGuest{val: counter}, nil
	})
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// assign for scheduling
	g1, err := pl.getForScheduling(ctx, id)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// reassign for binding
	pl.getForBinding(id)

	if pl.schedulingCycleID != 0 {
		t.Fatalf("expected no scheduling cycles")
	}

	if pl.scheduled != nil {
		t.Fatalf("expected no scheduling cycles")
	}

	// assign another for scheduling
	g2, err := pl.getForScheduling(ctx, differentID)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// reassign it for binding
	pl.getForBinding(differentID)

	if want, have := map[uint32]*testGuest{id: g1, differentID: g2}, pl.binding; !reflect.DeepEqual(want, have) {
		t.Fatalf("expected two guests in the binding cycle: want %v, have %v", want, have)
	}
}
