package field

import (
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func seedCorpus(f *testing.F) {
	seeds := [][]byte{
		make([]byte, 64),
		bytesRepeat(0xFF, 64),
	}
	// p in both halves.
	pb := bigToBytes(fieldPrime)
	pp := make([]byte, 64)
	copy(pp[:32], pb[:])
	copy(pp[32:], pb[:])
	seeds = append(seeds, pp)
	for _, s := range seeds {
		f.Add(s)
	}
}

func bytesRepeat(v byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}

func split2(data []byte) (a, b [32]byte, ok bool) {
	if len(data) < 64 {
		return a, b, false
	}
	copy(a[:], data[:32])
	copy(b[:], data[32:64])
	return a, b, true
}

func FuzzMul(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		a, b, ok := split2(data)
		if !ok {
			return
		}
		var va, vb, vr Val
		va.SetBytes(&a)
		vb.SetBytes(&b)
		vr.Mul2(&va, &vb).Normalize()

		var fa, fb, fr secp.FieldVal
		fa.SetBytes(&a)
		fb.SetBytes(&b)
		fr.Mul2(&fa, &fb).Normalize()

		if *vr.Bytes() != *fr.Bytes() {
			t.Fatalf("Mul mismatch a=%x b=%x", a, b)
		}
	})
}

func FuzzSquare(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 32 {
			return
		}
		var a [32]byte
		copy(a[:], data[:32])

		var va, vr Val
		va.SetBytes(&a)
		vr.SquareVal(&va).Normalize()

		var fa, fr secp.FieldVal
		fa.SetBytes(&a)
		fr.SquareVal(&fa).Normalize()

		if *vr.Bytes() != *fr.Bytes() {
			t.Fatalf("Square mismatch a=%x", a)
		}
	})
}

func FuzzInverse(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 32 {
			return
		}
		var a [32]byte
		copy(a[:], data[:32])

		var va, vr Val
		va.SetBytes(&a)
		vr.Set(&va).Inverse().Normalize()

		var fa, fr secp.FieldVal
		fa.SetBytes(&a)
		fr.Set(&fa).Inverse().Normalize()

		if *vr.Bytes() != *fr.Bytes() {
			t.Fatalf("Inverse mismatch a=%x", a)
		}

		// a * inv(a) == 1 for nonzero a.
		var chk Val
		chk.Set(&va).Normalize()
		if !chk.IsZero() {
			var prod Val
			prod.Mul2(&va, &vr).Normalize()
			if !prod.IsOne() {
				t.Fatalf("a*inv(a) != 1 for a=%x", a)
			}
		}
	})
}

func FuzzAddNegate(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		a, b, ok := split2(data)
		if !ok {
			return
		}
		var va, vb, vr Val
		va.SetBytes(&a)
		vb.SetBytes(&b)
		// (a + b) then negate.
		vr.Add2(&va, &vb)
		vr.Negate(2)
		vr.Normalize()

		var fa, fb, fr secp.FieldVal
		fa.SetBytes(&a)
		fb.SetBytes(&b)
		fr.Add2(&fa, &fb)
		fr.Negate(2)
		fr.Normalize()

		if *vr.Bytes() != *fr.Bytes() {
			t.Fatalf("AddNegate mismatch a=%x b=%x", a, b)
		}
	})
}
