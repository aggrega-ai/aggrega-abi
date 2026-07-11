// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"os"
	"strings"
	"testing"
)

// TestGeneratedCodec_ReflectFree asserts the generated codec source imports no
// reflect package and references no reflect symbol. The reflect-free property
// is the robustness claim of the generated codec — a guest decoding through it
// builds correctly without the GC sensitivity reflection drags in — so a
// regression that reintroduces reflect (via a future generator change) must
// fail loudly. The check is precise: it ignores the word "reflect" inside the
// "reflect-free" descriptor that appears in the generated godoc, matching only
// an actual import line or a reflect.Symbol reference.
func TestGeneratedCodec_ReflectFree(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"frp_gen.go", "cli_gen.go", "search_gen.go", "mutator_gen.go", "linter_gen.go", "ir_gen.go", "policy_gen.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read generated codec %s: %v", file, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			// An import of the reflect package, in either single- or
			// grouped-import form.
			if trimmed == `import "reflect"` || trimmed == `"reflect"` {
				t.Errorf("%s:%d imports reflect: %q", file, i+1, trimmed)
			}
			// A qualified reflect.* reference. Skip comment lines so the
			// "reflect-free" descriptor in the godoc does not trip the check.
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, "reflect.") {
				t.Errorf("%s:%d references reflect.: %q", file, i+1, trimmed)
			}
		}
	}
}
