//go:build arm64 && !purego

package field

import (
	"math/rand"
	"testing"
)

// TestAsmMatchesGenericARM64 pins the arm64 kernel to the portable backend
// bit-for-bit, independent of which backend the package selected at init.
func TestAsmMatchesGenericARM64(t *testing.T) {
	runAsmVsGeneric(t, mulArm64, sqrArm64)
}

func TestAsmRepeatedSquareARM64(t *testing.T) {
	rng := rand.New(rand.NewSource(0x53514e))
	for magnitude := 1; magnitude <= 8; magnitude++ {
		for i := 0; i < 256; i++ {
			v, _ := buildMag(rng, magnitude)
			for _, count := range []uint64{1, 2, 3, 5, 11, 22, 23, 44, 88, 255} {
				var want, got Val
				sqrGeneric(&want.n, &v.n)
				for j := uint64(1); j < count; j++ {
					sqrGeneric(&want.n, &want.n)
				}
				sqrArm64N(&got.n, &v.n, count)
				if got.n != want.n {
					t.Fatalf("repeated square mag=%d count=%d: got %v want %v", magnitude, count, got.n, want.n)
				}
				got = v
				sqrArm64N(&got.n, &got.n, count)
				if got.n != want.n {
					t.Fatalf("repeated square alias mag=%d count=%d: got %v want %v", magnitude, count, got.n, want.n)
				}
			}
		}
	}
}
