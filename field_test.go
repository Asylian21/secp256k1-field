package field

import (
	"math/big"
	"math/rand"
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// fieldPrime is p = 2^256 - 2^32 - 977.
var fieldPrime = func() *big.Int {
	p := new(big.Int).Lsh(big.NewInt(1), 256)
	p.Sub(p, new(big.Int).Lsh(big.NewInt(1), 32))
	p.Sub(p, big.NewInt(977))
	return p
}()

func bigToBytes(x *big.Int) *[32]byte {
	var b [32]byte
	x.FillBytes(b[:]) // big-endian, left-zero-padded; ignores anything above 256 bits
	return &b
}

// boundaryVectors returns the interesting edge values: 0, 1, 2, small constants,
// the prime and its neighbors, limb boundaries, and the all-ones overflow value.
func boundaryVectors() []*[32]byte {
	one := big.NewInt(1)
	mk := func(x *big.Int) *big.Int { return new(big.Int).Set(x) }
	vals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(977),
		big.NewInt(978),
		new(big.Int).Add(new(big.Int).Lsh(one, 32), big.NewInt(977)), // 2^32+977
		new(big.Int).Sub(fieldPrime, big.NewInt(2)),                  // p-2
		new(big.Int).Sub(fieldPrime, one),                            // p-1
		mk(fieldPrime),                                               // p  -> 0
		new(big.Int).Add(fieldPrime, one),                            // p+1 -> 1
		new(big.Int).Sub(new(big.Int).Lsh(one, 256), one),            // 2^256-1
		new(big.Int).Sub(new(big.Int).Lsh(one, 256), big.NewInt(2)),  // 2^256-2
		new(big.Int).Lsh(one, 52),                                    // limb boundary
		new(big.Int).Sub(new(big.Int).Lsh(one, 52), one),
		new(big.Int).Lsh(one, 104),
		new(big.Int).Lsh(one, 156),
		new(big.Int).Lsh(one, 208),
		new(big.Int).Sub(new(big.Int).Lsh(one, 256), new(big.Int).Lsh(one, 32)), // 2^256-2^32
		new(big.Int).Lsh(one, 255),
	}
	out := make([]*[32]byte, len(vals))
	for i, v := range vals {
		out[i] = bigToBytes(v)
	}
	return out
}

func TestBoundaryVectors(t *testing.T) {
	vecs := boundaryVectors()
	for ai, ab := range vecs {
		// SetBytes overflow flag.
		var va Val
		var fa secp.FieldVal
		if va.SetBytes(ab) != fa.SetBytes(ab) {
			t.Fatalf("vec %d: overflow flag mismatch", ai)
		}
		normEq(t, "boundary setbytes", &va, &fa)

		// Square and Inverse (unary).
		{
			va, fa := pair(ab)
			var vr Val
			vr.SquareVal(&va)
			var fr secp.FieldVal
			fr.SquareVal(&fa)
			normEq(t, "boundary square", &vr, &fr)
		}
		{
			va, fa := pair(ab)
			var vr Val
			vr.Set(&va).Inverse()
			var fr secp.FieldVal
			fr.Set(&fa).Inverse()
			normEq(t, "boundary inverse", &vr, &fr)
		}
		{
			va, fa := pair(ab)
			var vr Val
			vr.NegateVal(&va, 1)
			var fr secp.FieldVal
			fr.NegateVal(&fa, 1)
			normEq(t, "boundary negate", &vr, &fr)
		}

		// Binary ops against every other boundary value.
		for _, bb := range vecs {
			va, fa := pair(ab)
			vb, fb := pair(bb)
			var vr Val
			vr.Mul2(&va, &vb)
			var fr secp.FieldVal
			fr.Mul2(&fa, &fb)
			normEq(t, "boundary mul", &vr, &fr)

			var vra Val
			vra.Add2(&va, &vb)
			var fra secp.FieldVal
			fra.Add2(&fa, &fb)
			normEq(t, "boundary add", &vra, &fra)
		}
	}
}

func TestPredicatesMatchOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(0xc0ffee))
	check := func(b *[32]byte) {
		va, fa := pair(b)
		va.Normalize()
		fa.Normalize()
		if (va.IsZero() != fa.IsZero()) || (va.IsZeroBit() != fa.IsZeroBit()) {
			t.Fatalf("IsZero mismatch for %x", b[:])
		}
		if (va.IsOne() != fa.IsOne()) || (va.IsOneBit() != fa.IsOneBit()) {
			t.Fatalf("IsOne mismatch for %x", b[:])
		}
		if (va.IsOdd() != fa.IsOdd()) || (va.IsOddBit() != fa.IsOddBit()) {
			t.Fatalf("IsOdd mismatch for %x", b[:])
		}
		vb, fb := pair(b)
		vb.Normalize()
		fb.Normalize()
		if va.Equals(&vb) != fa.Equals(&fb) {
			t.Fatalf("Equals mismatch for %x", b[:])
		}
	}
	for _, b := range boundaryVectors() {
		check(b)
	}
	for i := 0; i < 50000; i++ {
		check(randBytes(rng))
	}
}

func TestSetIntAndSetByteSlice(t *testing.T) {
	for _, n := range []uint16{0, 1, 2, 977, 65535} {
		var v Val
		var f secp.FieldVal
		v.SetInt(n)
		f.SetInt(n)
		normEq(t, "setint", &v, &f)
	}
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 10000; i++ {
		b := randBytes(rng)
		var v Val
		var f secp.FieldVal
		gotOver := v.SetByteSlice(b[:])
		wantOver := f.SetByteSlice(b[:])
		if gotOver != wantOver {
			t.Fatalf("SetByteSlice overflow mismatch")
		}
		normEq(t, "setbyteslice", &v, &f)
	}
}

func TestZeroAllocations(t *testing.T) {
	var a, b, r Val
	rng := rand.New(rand.NewSource(99))
	a.SetBytes(randBytes(rng))
	b.SetBytes(randBytes(rng))
	var buf [32]byte

	cases := []struct {
		name string
		fn   func()
	}{
		{"Mul2", func() { r.Mul2(&a, &b) }},
		{"SquareVal", func() { r.SquareVal(&a) }},
		{"Add2", func() { r.Add2(&a, &b) }},
		{"NegateVal", func() { r.NegateVal(&a, 1) }},
		{"Normalize", func() { r.Set(&a); r.Normalize() }},
		{"SetBytes", func() { r.SetBytes(&buf) }},
		{"PutBytesUnchecked", func() { a.Normalize(); a.PutBytesUnchecked(buf[:]) }},
		{"Inverse", func() { r.Set(&a); r.Inverse() }},
	}
	for _, c := range cases {
		if n := testing.AllocsPerRun(200, c.fn); n != 0 {
			t.Errorf("%s: got %v allocs/op, want 0", c.name, n)
		}
	}
}

func TestBackendReported(t *testing.T) {
	if got := Backend(); got == "" {
		t.Fatal("Backend() returned empty string")
	}
	// Smoke-test that the active backend actually computes: 2 * 3 == 6.
	var a, b, r Val
	a.SetInt(2)
	b.SetInt(3)
	r.Mul2(&a, &b).Normalize()
	if !r.Equals(new(Val).SetInt(6)) {
		t.Fatalf("active backend %q computed 2*3 = %s, want 6", Backend(), r.String())
	}
}
