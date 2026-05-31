package field

import (
	"math/rand"
	"testing"
)

// runAsmVsGeneric checks that an assembler kernel produces byte-identical limb
// output to the portable backend (a stronger property than equal-after-Normalize)
// across random magnitude-1..8 inputs, and that it tolerates output/input
// aliasing the way the consumer relies on. It is invoked from the
// architecture-specific test entry points.
func runAsmVsGeneric(t *testing.T, mul func(r, a, b *[5]uint64), sqr func(r, a *[5]uint64)) {
	t.Helper()
	rng := rand.New(rand.NewSource(0x4a4a4a))
	for i := 0; i < 300000; i++ {
		magA := 1 + rng.Intn(8)
		magB := 1 + rng.Intn(8)
		va, _ := buildMag(rng, magA)
		vb, _ := buildMag(rng, magB)

		var g, a Val
		mulGeneric(&g.n, &va.n, &vb.n)
		mul(&a.n, &va.n, &vb.n)
		if g.n != a.n {
			t.Fatalf("mul asm != generic\n a=%v\n b=%v\n gen=%v\n asm=%v", va.n, vb.n, g.n, a.n)
		}

		sqrGeneric(&g.n, &va.n)
		sqr(&a.n, &va.n)
		if g.n != a.n {
			t.Fatalf("sqr asm != generic\n a=%v\n gen=%v\n asm=%v", va.n, g.n, a.n)
		}
	}

	// Aliasing: r == a, r == b, and square in place (r == a).
	va, _ := buildMag(rng, 3)
	vb, _ := buildMag(rng, 5)
	var want Val
	mulGeneric(&want.n, &va.n, &vb.n)

	ra := va
	mul(&ra.n, &ra.n, &vb.n)
	if ra.n != want.n {
		t.Fatalf("mul aliasing r==a mismatch: got %v want %v", ra.n, want.n)
	}
	rb := vb
	mul(&rb.n, &va.n, &rb.n)
	if rb.n != want.n {
		t.Fatalf("mul aliasing r==b mismatch: got %v want %v", rb.n, want.n)
	}

	var wantSq Val
	sqrGeneric(&wantSq.n, &va.n)
	rs := va
	sqr(&rs.n, &rs.n)
	if rs.n != wantSq.n {
		t.Fatalf("sqr aliasing r==a mismatch: got %v want %v", rs.n, wantSq.n)
	}
}
