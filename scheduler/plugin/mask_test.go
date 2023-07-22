package wasm

import (
	"testing"
)

// Test_maskInterfaces tests a few combinations. Since there are thousands of
// combinations of interfaces, a fuzz test would be better.
func Test_maskInterfaces(t *testing.T) {
	tests := []struct {
		name          string
		plugin        *wasmPlugin
		expectError   bool
		expectFilter  bool
		expectScore   bool
		expectReserve bool
		expectPermit  bool
		expectBind    bool
	}{
		{
			name:        "prescore",
			plugin:      &wasmPlugin{guestInterfaces: iPreScorePlugin},
			expectError: true, // not supported to prescore without score
		},
		{
			name:   "prefilter", // special case of filter
			plugin: &wasmPlugin{guestInterfaces: iPreFilterPlugin},
		},
		{
			name:         "prefilter|filter",
			plugin:       &wasmPlugin{guestInterfaces: iPreFilterPlugin | iFilterPlugin},
			expectFilter: true,
		},
		{
			name:         "filter",
			plugin:       &wasmPlugin{guestInterfaces: iFilterPlugin},
			expectFilter: true,
		},
		{
			name:        "prefilter|score",
			plugin:      &wasmPlugin{guestInterfaces: iPreFilterPlugin | iScorePlugin},
			expectScore: true,
		},
		{
			name:        "prescore|score",
			plugin:      &wasmPlugin{guestInterfaces: iPreScorePlugin | iScorePlugin},
			expectScore: true,
		},
		{
			name:        "score",
			plugin:      &wasmPlugin{guestInterfaces: iScorePlugin},
			expectScore: true,
		},
		{
			name:         "prefilter|filter|score",
			plugin:       &wasmPlugin{guestInterfaces: iPreFilterPlugin | iFilterPlugin | iScorePlugin},
			expectFilter: true,
			expectScore:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := maskInterfaces(tc.plugin)
			if tc.expectError {
				if err == nil {
					t.Fatal("expected to error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if _, ok := p.(basePlugin); !ok {
				t.Fatalf("expected basePlugin %v", p)
			}
			if _, ok := p.(filterPlugin); tc.expectFilter != ok {
				t.Fatalf("didn't expect filterPlugin %v", p)
			}
			if _, ok := p.(scorePlugin); tc.expectScore != ok {
				t.Fatalf("didn't expect scorePlugin %v", p)
			}
			if _, ok := p.(reservePlugin); tc.expectReserve != ok {
				t.Fatalf("didn't expect reservePlugin %v", p)
			}
			if _, ok := p.(permitPlugin); tc.expectPermit != ok {
				t.Fatalf("didn't expect permitPlugin %v", p)
			}
			if _, ok := p.(bindPlugin); tc.expectBind != ok {
				t.Fatalf("didn't expect bindPlugin %v", p)
			}
		})
	}
}
