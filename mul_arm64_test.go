//go:build arm64 && !purego

package field

import (
	"math/rand"
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// TestAsmMatchesGenericARM64 pins the arm64 kernel to the portable backend
// bit-for-bit, independent of which backend the package selected at init.
func TestAsmMatchesGenericARM64(t *testing.T) {
	runAsmVsGeneric(t, mulArm64, sqrArm64)
}

func arm64MagnitudeBoundaryLimbs(magnitude uint64) [][5]uint64 {
	max := [5]uint64{
		2 * magnitude * limbMask, 2 * magnitude * limbMask,
		2 * magnitude * limbMask, 2 * magnitude * limbMask,
		2 * magnitude * limb4Mask,
	}
	vectors := [][5]uint64{
		{}, {1}, max,
		{max[0], 0, max[2], 0, max[4]},
		{0, max[1], 0, max[3], 0},
		{p0, limbMask, limbMask, limbMask, limb4Mask}, // zero modulo p
	}
	for limb := range max {
		var isolated [5]uint64
		isolated[limb] = max[limb]
		vectors = append(vectors, isolated)
	}
	return vectors
}

func TestAsmMagnitudeBoundariesARM64(t *testing.T) {
	for magnitude := uint64(1); magnitude <= 8; magnitude++ {
		vectors := arm64MagnitudeBoundaryLimbs(magnitude)
		for ai, a := range vectors {
			canonical := Val{n: a}
			canonical.Normalize()
			var fa secp.FieldVal
			fa.SetBytes(canonical.Bytes())
			var got, want Val
			sqrGeneric(&want.n, &a)
			sqrArm64(&got.n, &a)
			if got.n != want.n {
				t.Fatalf("square boundary mag=%d vector=%d: got %v want %v", magnitude, ai, got.n, want.n)
			}
			inPlace := Val{n: a}
			sqrArm64(&inPlace.n, &inPlace.n)
			if inPlace.n != want.n {
				t.Fatalf("square boundary alias mag=%d vector=%d: got %v want %v", magnitude, ai, inPlace.n, want.n)
			}
			oracle := fa
			oracle.Square()
			normEq(t, "square magnitude boundary", &got, &oracle)
			for bi, b := range vectors {
				mulGeneric(&want.n, &a, &b)
				mulArm64(&got.n, &a, &b)
				if got.n != want.n {
					t.Fatalf("mul boundary mag=%d vectors=%d,%d: got %v want %v", magnitude, ai, bi, got.n, want.n)
				}
				aliasA, aliasB := Val{n: a}, Val{n: b}
				mulArm64(&aliasA.n, &aliasA.n, &b)
				mulArm64(&aliasB.n, &a, &aliasB.n)
				if aliasA.n != want.n || aliasB.n != want.n {
					t.Fatalf("mul boundary alias mag=%d vectors=%d,%d: got %v / %v want %v", magnitude, ai, bi, aliasA.n, aliasB.n, want.n)
				}
				canonical.n = b
				canonical.Normalize()
				var fb secp.FieldVal
				fb.SetBytes(canonical.Bytes())
				oracle.Mul2(&fa, &fb)
				normEq(t, "mul magnitude boundary", &got, &oracle)
			}
		}
	}
}

// Sum-built random inputs tend to leave ample headroom in each limb. Exercise
// the actual magnitude limits as well, including the initial wide operands,
// every fixed inverse-chain count, and in-place output register reuse.
func TestAsmRepeatedSquareBoundariesARM64(t *testing.T) {
	for magnitude := uint64(1); magnitude <= 8; magnitude++ {
		vectors := arm64MagnitudeBoundaryLimbs(magnitude)
		for vi, limbs := range vectors {
			input := Val{n: limbs}
			canonical := input
			canonical.Normalize()
			for _, count := range []uint64{1, 2, 3, 5, 11, 22, 23, 44, 88, 255} {
				var want Val
				sqrGeneric(&want.n, &input.n)
				for j := uint64(1); j < count; j++ {
					sqrGeneric(&want.n, &want.n)
				}
				var got Val
				sqrArm64N(&got.n, &input.n, count)
				if got.n != want.n {
					t.Fatalf("boundary mag=%d vector=%d count=%d: got %v want %v", magnitude, vi, count, got.n, want.n)
				}
				if input.n != limbs {
					t.Fatalf("boundary mag=%d vector=%d count=%d: input modified", magnitude, vi, count)
				}
				inPlace := input
				sqrArm64N(&inPlace.n, &inPlace.n, count)
				if inPlace.n != want.n {
					t.Fatalf("boundary alias mag=%d vector=%d count=%d: got %v want %v", magnitude, vi, count, inPlace.n, want.n)
				}
				var oracle secp.FieldVal
				oracle.SetBytes(canonical.Bytes())
				for j := uint64(0); j < count; j++ {
					oracle.Square()
				}
				normEq(t, "boundary repeated square", &got, &oracle)
			}
		}
	}
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
