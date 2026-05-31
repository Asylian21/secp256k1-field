package field

import (
	"math/rand"
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// The dcrd FieldVal is the differential-correctness oracle: for every operation
// and every input, the normalized 32-byte output of this package must be
// bit-identical to dcrd's. dcrd is imported only by tests.

// randBytes returns 32 random bytes. Any 256-bit value is a valid SetBytes
// input for both implementations (values >= p are reduced identically on
// Normalize), so no explicit reduction is needed.
func randBytes(rng *rand.Rand) *[32]byte {
	var b [32]byte
	rng.Read(b[:])
	return &b
}

// pair sets a fresh Val and dcrd FieldVal from the same bytes.
func pair(b *[32]byte) (Val, secp.FieldVal) {
	var v Val
	var f secp.FieldVal
	v.SetBytes(b)
	f.SetBytes(b)
	return v, f
}

// buildMag returns a Val and dcrd FieldVal that hold the same value at the given
// magnitude (1..8), constructed as a sum of `mag` normalized random terms so the
// magnitude bookkeeping matches in both representations.
func buildMag(rng *rand.Rand, mag int) (Val, secp.FieldVal) {
	v, f := pair(randBytes(rng))
	for i := 1; i < mag; i++ {
		v2, f2 := pair(randBytes(rng))
		v.Add(&v2)
		f.Add(&f2)
	}
	return v, f
}

// normEq fails the test unless got (a Val) and want (a dcrd FieldVal) are equal
// after normalizing both.
func normEq(t *testing.T, ctx string, got *Val, want *secp.FieldVal) {
	t.Helper()
	got.Normalize()
	want.Normalize()
	gb := got.Bytes()
	wb := want.Bytes()
	if *gb != *wb {
		t.Fatalf("%s mismatch:\n got  %x\n want %x", ctx, gb[:], wb[:])
	}
}

const diffIters = 200000

func TestDiffSetBytes(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5e1f))
	for i := 0; i < diffIters; i++ {
		b := randBytes(rng)
		var v Val
		var f secp.FieldVal
		gotOverflow := v.SetBytes(b)
		wantOverflow := f.SetBytes(b)
		if gotOverflow != wantOverflow {
			t.Fatalf("SetBytes overflow flag mismatch for %x: got %d want %d", b[:], gotOverflow, wantOverflow)
		}
		normEq(t, "SetBytes", &v, &f)
	}
}

func TestDiffRoundTripBytes(t *testing.T) {
	rng := rand.New(rand.NewSource(0xb172))
	for i := 0; i < diffIters; i++ {
		b := randBytes(rng)
		var v Val
		v.SetBytes(b)
		v.Normalize()
		// Round-trip through bytes must be stable.
		out := v.Bytes()
		var v2 Val
		v2.SetBytes(out)
		v2.Normalize()
		if !v.Equals(&v2) {
			t.Fatalf("byte round-trip unstable: %x -> %x", b[:], out[:])
		}
	}
}

func TestDiffAdd(t *testing.T) {
	rng := rand.New(rand.NewSource(0xadd))
	for i := 0; i < diffIters; i++ {
		magA := 1 + rng.Intn(8)
		magB := 1 + rng.Intn(9-magA) // keep magA+magB <= 9 well within bounds
		va, fa := buildMag(rng, magA)
		vb, fb := buildMag(rng, magB)
		var vr Val
		vr.Add2(&va, &vb)
		var fr secp.FieldVal
		fr.Add2(&fa, &fb)
		normEq(t, "Add2", &vr, &fr)
	}
}

func TestDiffNegate(t *testing.T) {
	rng := rand.New(rand.NewSource(0x9e6a7e))
	for i := 0; i < diffIters; i++ {
		mag := 1 + rng.Intn(8)
		va, fa := buildMag(rng, mag)
		var vr Val
		vr.NegateVal(&va, uint32(mag))
		var fr secp.FieldVal
		fr.NegateVal(&fa, uint32(mag))
		normEq(t, "NegateVal", &vr, &fr)
	}
}

func TestDiffMul(t *testing.T) {
	rng := rand.New(rand.NewSource(0x33))
	for i := 0; i < diffIters; i++ {
		magA := 1 + rng.Intn(8)
		magB := 1 + rng.Intn(8)
		va, fa := buildMag(rng, magA)
		vb, fb := buildMag(rng, magB)

		var vr Val
		vr.Mul2(&va, &vb)
		var fr secp.FieldVal
		fr.Mul2(&fa, &fb)
		normEq(t, "Mul2", &vr, &fr)

		// Direct generic backend check (pins generic even when the active
		// backend is assembler).
		var gr Val
		mulGeneric(&gr.n, &va.n, &vb.n)
		normEq(t, "mulGeneric", &gr, &fr)
	}
}

func TestDiffSquare(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5a))
	for i := 0; i < diffIters; i++ {
		mag := 1 + rng.Intn(8)
		va, fa := buildMag(rng, mag)

		var vr Val
		vr.SquareVal(&va)
		var fr secp.FieldVal
		fr.SquareVal(&fa)
		normEq(t, "SquareVal", &vr, &fr)

		var gr Val
		sqrGeneric(&gr.n, &va.n)
		normEq(t, "sqrGeneric", &gr, &fr)
	}
}

func TestDiffInverse(t *testing.T) {
	rng := rand.New(rand.NewSource(0x142e35e))
	const iters = 8000
	for i := 0; i < iters; i++ {
		va, fa := pair(randBytes(rng))
		var vr Val
		vr.Set(&va).Inverse()
		var fr secp.FieldVal
		fr.Set(&fa).Inverse()
		normEq(t, "Inverse", &vr, &fr)

		// Self-consistency: a * a^-1 == 1 (for nonzero a).
		var chk Val
		chk.Set(&va).Normalize()
		if !chk.IsZero() {
			var prod Val
			prod.Mul2(&va, &vr).Normalize()
			if !prod.IsOne() {
				t.Fatalf("a*inv(a) != 1 for a=%s", va.String())
			}
		}
	}
}

// TestDiffHotLoopShape replays the exact field-operation/magnitude sequence the
// consumer's batched affine point addition performs, comparing against dcrd at
// every materialized coordinate. This is the most representative differential
// test for the integration target.
func TestDiffHotLoopShape(t *testing.T) {
	rng := rand.New(rand.NewSource(0x60710092))
	for i := 0; i < 40000; i++ {
		// Inputs: base point (px,py) and an addend (gx,gy), all normalized.
		vpx, fpx := pair(randBytes(rng))
		vpy, fpy := pair(randBytes(rng))
		vgx, fgx := pair(randBytes(rng))
		vgy, fgy := pair(randBytes(rng))
		vpx.Normalize()
		fpx.Normalize()
		vpy.Normalize()
		fpy.Normalize()
		vgx.Normalize()
		fgx.Normalize()
		vgy.Normalize()
		fgy.Normalize()

		// negPx, negPy (mag 2)
		var vNegPx, vNegPy Val
		vNegPx.NegateVal(&vpx, 1)
		vNegPy.NegateVal(&vpy, 1)
		var fNegPx, fNegPy secp.FieldVal
		fNegPx.NegateVal(&fpx, 1)
		fNegPy.NegateVal(&fpy, 1)

		// dx = gx + negPx (mag 3); skip degenerate dx==0.
		var vdx Val
		vdx.Add2(&vgx, &vNegPx)
		var fdx secp.FieldVal
		fdx.Add2(&fgx, &fNegPx)
		var dxChk Val
		dxChk.Set(&vdx).Normalize()
		if dxChk.IsZero() {
			continue
		}

		// 1/dx
		var vInv Val
		vInv.Set(&vdx).Inverse()
		var fInv secp.FieldVal
		fInv.Set(&fdx).Inverse()

		// num = gy + negPy (mag 3); lam = num/dx (mag 1)
		var vNum, vLam Val
		vNum.Add2(&vgy, &vNegPy)
		vLam.Mul2(&vNum, &vInv)
		var fNum, fLam secp.FieldVal
		fNum.Add2(&fgy, &fNegPy)
		fLam.Mul2(&fNum, &fInv)

		// lamSq (mag 1); x3 = lamSq + negPx + negGx (mag 4)
		var vLamSq, vNegGx, vX3 Val
		vLamSq.SquareVal(&vLam)
		vNegGx.NegateVal(&vgx, 1)
		vX3.Set(&vLamSq).Add(&vNegPx).Add(&vNegGx)
		var fLamSq, fNegGx, fX3 secp.FieldVal
		fLamSq.SquareVal(&fLam)
		fNegGx.NegateVal(&fgx, 1)
		fX3.Set(&fLamSq).Add(&fNegPx).Add(&fNegGx)

		// negX3 = -x3 (mag 5); t = px + negX3 (mag 6)
		var vNegX3, vT Val
		vNegX3.NegateVal(&vX3, 4)
		vT.Add2(&vpx, &vNegX3)
		var fNegX3, fT secp.FieldVal
		fNegX3.NegateVal(&fX3, 4)
		fT.Add2(&fpx, &fNegX3)

		// y3 = lam*t + negPy (mag 3)
		var vY3 Val
		vY3.Mul2(&vLam, &vT).Add(&vNegPy)
		var fY3 secp.FieldVal
		fY3.Mul2(&fLam, &fT).Add(&fNegPy)

		normEq(t, "hotloop x3", &vX3, &fX3)
		normEq(t, "hotloop y3", &vY3, &fY3)
	}
}
