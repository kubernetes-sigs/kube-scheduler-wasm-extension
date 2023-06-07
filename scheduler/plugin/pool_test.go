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
	"testing"

	"k8s.io/apimachinery/pkg/types"
)

var ctx = context.Background()

type testGuest struct {
	val int
}

func Test_guestPool_getOrCreateGuest(t *testing.T) {
	uid := types.UID("test-uid")
	differentUID := types.UID("different-uid")

	var counter int
	pl, err := newGuestPool(ctx, func(ctx2 context.Context) (*testGuest, error) {
		counter++
		return &testGuest{val: counter}, nil
	})
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	g1, err := pl.getOrCreateGuest(ctx, uid)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g1 == nil {
		t.Fatalf("have nil guest instance")
	}

	// We expect a new guest instance when created with a different podUID.
	g2, err := pl.getOrCreateGuest(ctx, differentUID)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g2 == nil {
		t.Fatalf("have nil guest instance")
	}
	if g2 == g1 {
		t.Fatalf("expected different guests, but they are the same")
	}

	// Put the first back into the pool
	pl.put(g1)

	// This should return the first guest instance because we pass the same
	// podUID.
	g3, err := pl.getOrCreateGuest(ctx, uid)
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g3 != g1 {
		t.Fatalf("unexpected guest: want %v, have %v", g1, g3)
	}
}
