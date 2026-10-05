package field

import (
	"encoding/binary"
	"encoding/hex"
)

// Field representation constants.
const (
	// limbBits is the number of value bits stored per limb.
	limbBits = 52

	// limbMask is the mask for a full 52-bit limb (2^52 - 1).
	limbMask = 0xFFFFFFFFFFFFF

	// limb4Mask is the mask for the top limb, which holds the highest 48 bits
	// of the 256-bit value (2^48 - 1).
	limb4Mask = 0x0FFFFFFFFFFFF

	// pReduce is 2^256 mod p = 2^32 + 977. Adding pReduce*x folds a stray
	// multiple x of 2^256 back into the low limbs during normalization.
	pReduce = 0x1000003D1

	// p0 is the lowest limb of the field prime p in the 5x52 layout
	// (p = 2^256 - 2^32 - 977). Used by Normalize's final-reduction test and
	// by negation as the limb-0 minuend.
	p0 = 0xFFFFEFFFFFC2F
)

// Val is an element of the secp256k1 base field Fp (integers modulo
// p = 2^256 - 2^32 - 977), stored as five base-2^52 limbs.
//
// Like the dcrd FieldVal it mirrors, Val performs NO validation of the
// normalization/magnitude preconditions documented on each method; satisfying
// them is the caller's responsibility. See the package documentation for the
// normalization and magnitude model.
type Val struct {
	// n holds the value as n[0] + n[1]*2^52 + n[2]*2^104 + n[3]*2^156 +
	// n[4]*2^208. A normalized value keeps n[0..3] < 2^52 and n[4] < 2^48.
	n [5]uint64
}

// Zero sets the field value to zero. A newly created Val is already zero.
//
// Output Normalized: Yes
// Output Max Magnitude: 1
func (f *Val) Zero() {
	f.n[0] = 0
	f.n[1] = 0
	f.n[2] = 0
	f.n[3] = 0
	f.n[4] = 0
}

// Set sets f equal to val and returns f to support chaining. The normalization
// and magnitude of f become identical to val.
//
// Output Normalized: Same as input
// Output Max Magnitude: Same as input
func (f *Val) Set(val *Val) *Val {
	*f = *val
	return f
}

// SetInt sets f to the small unsigned integer ui and returns f.
//
// Output Normalized: Yes
// Output Max Magnitude: 1
func (f *Val) SetInt(ui uint16) *Val {
	f.n[0] = uint64(ui)
	f.n[1] = 0
	f.n[2] = 0
	f.n[3] = 0
	f.n[4] = 0
	return f
}

// SetBytes packs the 32-byte big-endian value b into f and returns 1 if the
// value is greater than or equal to the field prime (it overflowed) or 0
// otherwise. The semantics match dcrd's FieldVal.SetBytes.
//
// Output Normalized: Yes if no overflow, no otherwise
// Output Max Magnitude: 1
func (f *Val) SetBytes(b *[32]byte) uint32 {
	// Wide loads let the compiler use native endian reversal instead of
	// assembling the same words with individual byte loads.
	w0 := binary.BigEndian.Uint64(b[24:32])
	w1 := binary.BigEndian.Uint64(b[16:24])
	w2 := binary.BigEndian.Uint64(b[8:16])
	w3 := binary.BigEndian.Uint64(b[0:8])
	f.n[0] = w0 & limbMask
	f.n[1] = (w0>>52 | w1<<12) & limbMask
	f.n[2] = (w1>>40 | w2<<24) & limbMask
	f.n[3] = (w2>>28 | w3<<36) & limbMask
	f.n[4] = w3 >> 16

	// The packed value is >= p exactly when the top four 52-bit limbs are
	// saturated to p's limbs and the low limb reaches p's low limb.
	var overflow uint32
	if f.n[4] == limb4Mask && (f.n[3]&f.n[2]&f.n[1]) == limbMask && f.n[0] >= p0 {
		overflow = 1
	}
	return overflow
}

// SetByteSlice interprets b as a big-endian unsigned integer (truncated to its
// low 32 bytes), packs it into f, and reports whether the truncated value is
// greater than or equal to the field prime.
//
// Output Normalized: Yes if no overflow, no otherwise
// Output Max Magnitude: 1
func (f *Val) SetByteSlice(b []byte) bool {
	var b32 [32]byte
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(b32[32-len(b):], b)
	return f.SetBytes(&b32) != 0
}

// PutBytesUnchecked writes f as a 32-byte big-endian value into b, which must
// have at least 32 bytes of room or PutBytesUnchecked panics.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) PutBytesUnchecked(b []byte) {
	_ = b[31]
	n0, n1, n2, n3, n4 := f.n[0], f.n[1], f.n[2], f.n[3], f.n[4]
	binary.BigEndian.PutUint64(b[0:8], n4<<16|n3>>36)
	binary.BigEndian.PutUint64(b[8:16], n3<<28|n2>>24)
	binary.BigEndian.PutUint64(b[16:24], n2<<40|n1>>12)
	binary.BigEndian.PutUint64(b[24:32], n1<<52|n0)
}

// PutBytes writes f as a 32-byte big-endian value into the array b.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) PutBytes(b *[32]byte) {
	f.PutBytesUnchecked(b[:])
}

// Bytes returns f as a freshly allocated 32-byte big-endian array.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) Bytes() *[32]byte {
	var b [32]byte
	f.PutBytesUnchecked(b[:])
	return &b
}

// String returns the normalized value as a 64-character hex string. It does not
// modify f.
func (f Val) String() string {
	f.Normalize()
	b := f.Bytes()
	return hex.EncodeToString(b[:])
}

// Normalize reduces f to its unique canonical representative in [0, p) and
// returns f.
//
// Output Normalized: Yes
// Output Max Magnitude: 1
func (f *Val) Normalize() *Val {
	t0, t1, t2, t3, t4 := f.n[0], f.n[1], f.n[2], f.n[3], f.n[4]

	// Fold the bits above 2^256 (the top of limb 4) back into the low limbs,
	// then propagate carries once. After this pass the magnitude is 1.
	x := t4 >> 48
	t4 &= limb4Mask
	t0 += x * pReduce
	t1 += t0 >> limbBits
	t0 &= limbMask
	t2 += t1 >> limbBits
	t1 &= limbMask
	m := t1
	t3 += t2 >> limbBits
	t2 &= limbMask
	m &= t2
	t4 += t3 >> limbBits
	t3 &= limbMask
	m &= t3

	// A single conditional final reduction handles a carry to bit 256 or a
	// value still in [p, 2^256).
	var fin uint64
	if t4 == limb4Mask && m == limbMask && t0 >= p0 {
		fin = 1
	}
	x = (t4 >> 48) | fin

	// Most public field values are already below p after the first carry
	// pass. Skip the second pass unless a final reduction is needed. Like
	// the existing comparison above, this deliberately is variable time.
	if x != 0 {
		t0 += x * pReduce
		t1 += t0 >> limbBits
		t0 &= limbMask
		t2 += t1 >> limbBits
		t1 &= limbMask
		t3 += t2 >> limbBits
		t2 &= limbMask
		t4 += t3 >> limbBits
		t3 &= limbMask
		t4 &= limb4Mask
	}

	f.n[0], f.n[1], f.n[2], f.n[3], f.n[4] = t0, t1, t2, t3, t4
	return f
}

// IsZeroBit returns 1 when f is zero and 0 otherwise.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsZeroBit() uint32 {
	if (f.n[0] | f.n[1] | f.n[2] | f.n[3] | f.n[4]) == 0 {
		return 1
	}
	return 0
}

// IsZero reports whether f is zero.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsZero() bool {
	return (f.n[0] | f.n[1] | f.n[2] | f.n[3] | f.n[4]) == 0
}

// IsOneBit returns 1 when f equals one and 0 otherwise.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsOneBit() uint32 {
	if (f.n[0]^1)|f.n[1]|f.n[2]|f.n[3]|f.n[4] == 0 {
		return 1
	}
	return 0
}

// IsOne reports whether f equals one.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsOne() bool {
	return (f.n[0]^1)|f.n[1]|f.n[2]|f.n[3]|f.n[4] == 0
}

// IsOddBit returns 1 when f is odd and 0 otherwise.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsOddBit() uint32 {
	return uint32(f.n[0] & 1)
}

// IsOdd reports whether f is odd.
//
// Preconditions:
//   - f MUST be normalized.
func (f *Val) IsOdd() bool {
	return f.n[0]&1 == 1
}

// Equals reports whether f and val represent the same field element.
//
// Preconditions:
//   - Both f and val MUST be normalized.
func (f *Val) Equals(val *Val) bool {
	return f.n == val.n
}

// Add adds val to f and returns f.
//
// Preconditions:
//   - The sum of the magnitudes of f and val MUST be at most 32.
//
// Output Normalized: No
// Output Max Magnitude: sum of the two input magnitudes
func (f *Val) Add(val *Val) *Val {
	f.n[0] += val.n[0]
	f.n[1] += val.n[1]
	f.n[2] += val.n[2]
	f.n[3] += val.n[3]
	f.n[4] += val.n[4]
	return f
}

// Add2 sets f to val + val2 and returns f.
//
// Preconditions:
//   - The sum of the magnitudes of val and val2 MUST be at most 32.
//
// Output Normalized: No
// Output Max Magnitude: sum of the two input magnitudes
func (f *Val) Add2(val, val2 *Val) *Val {
	f.n[0] = val.n[0] + val2.n[0]
	f.n[1] = val.n[1] + val2.n[1]
	f.n[2] = val.n[2] + val2.n[2]
	f.n[3] = val.n[3] + val2.n[3]
	f.n[4] = val.n[4] + val2.n[4]
	return f
}

// AddInt adds the small unsigned integer ui to f and returns f.
//
// Output Normalized: No
// Output Max Magnitude: f's magnitude + 1
func (f *Val) AddInt(ui uint16) *Val {
	f.n[0] += uint64(ui)
	return f
}

// MulInt multiplies every limb of f by the small integer val and returns f.
//
// Preconditions:
//   - f's magnitude times val MUST be at most 32.
//
// Output Normalized: No
// Output Max Magnitude: f's magnitude times val
func (f *Val) MulInt(val uint8) *Val {
	v := uint64(val)
	f.n[0] *= v
	f.n[1] *= v
	f.n[2] *= v
	f.n[3] *= v
	f.n[4] *= v
	return f
}

// NegateVal sets f to the negation of val and returns f. The caller must supply
// val's magnitude.
//
// Preconditions:
//   - magnitude MUST be at most 31.
//
// Output Normalized: No
// Output Max Magnitude: magnitude + 1
func (f *Val) NegateVal(val *Val, magnitude uint32) *Val {
	// -val = (multiple of p) - val. Using 2*(m+1)*p_limb as the minuend keeps
	// every limb non-negative for any input limb within magnitude m, and bumps
	// the magnitude by one.
	k := 2 * (uint64(magnitude) + 1)
	f.n[0] = k*p0 - val.n[0]
	f.n[1] = k*limbMask - val.n[1]
	f.n[2] = k*limbMask - val.n[2]
	f.n[3] = k*limbMask - val.n[3]
	f.n[4] = k*limb4Mask - val.n[4]
	return f
}

// Negate negates f in place and returns f. The caller must supply f's
// magnitude.
//
// Preconditions:
//   - magnitude MUST be at most 31.
//
// Output Normalized: No
// Output Max Magnitude: magnitude + 1
func (f *Val) Negate(magnitude uint32) *Val {
	return f.NegateVal(f, magnitude)
}

// Mul multiplies f by val and returns f.
//
// Preconditions:
//   - Both f and val MUST have magnitude at most 8.
//
// Output Normalized: No
// Output Max Magnitude: 1
func (f *Val) Mul(val *Val) *Val {
	fieldMul(&f.n, &f.n, &val.n)
	return f
}

// Mul2 sets f to val * val2 and returns f.
//
// Preconditions:
//   - Both val and val2 MUST have magnitude at most 8.
//
// Output Normalized: No
// Output Max Magnitude: 1
func (f *Val) Mul2(val, val2 *Val) *Val {
	fieldMul(&f.n, &val.n, &val2.n)
	return f
}

// Square squares f in place and returns f.
//
// Preconditions:
//   - f MUST have magnitude at most 8.
//
// Output Normalized: No
// Output Max Magnitude: 1
func (f *Val) Square() *Val {
	fieldSqr(&f.n, &f.n)
	return f
}

// SquareVal sets f to val^2 and returns f.
//
// Preconditions:
//   - val MUST have magnitude at most 8.
//
// Output Normalized: No
// Output Max Magnitude: 1
func (f *Val) SquareVal(val *Val) *Val {
	fieldSqr(&f.n, &val.n)
	return f
}

// Inverse sets f to its modular multiplicative inverse (f^(p-2) mod p) and
// returns f. If f is zero the result is zero.
//
// Preconditions:
//   - f MUST have magnitude at most 8.
//
// Output Normalized: No
// Output Max Magnitude: 1
func (f *Val) Inverse() *Val {
	// Build x_n = f^(2^n-1), then append the low bits of p-2. This fixed
	// chain needs 255 squarings and 15 multiplications, with no lookup table.
	var x2, x3, x6, x9, x11, x22, x44, x88, x176, x220, x223 Val
	x2.SquareVal(f).Mul(f)
	x3.SquareVal(&x2).Mul(f)
	fieldSqrN(&x6.n, &x3.n, 3)
	x6.Mul(&x3)
	fieldSqrN(&x9.n, &x6.n, 3)
	x9.Mul(&x3)
	fieldSqrN(&x11.n, &x9.n, 2)
	x11.Mul(&x2)
	fieldSqrN(&x22.n, &x11.n, 11)
	x22.Mul(&x11)
	fieldSqrN(&x44.n, &x22.n, 22)
	x44.Mul(&x22)
	fieldSqrN(&x88.n, &x44.n, 44)
	x88.Mul(&x44)
	fieldSqrN(&x176.n, &x88.n, 88)
	x176.Mul(&x88)
	fieldSqrN(&x220.n, &x176.n, 44)
	x220.Mul(&x44)
	fieldSqrN(&x223.n, &x220.n, 3)
	x223.Mul(&x3)

	// (((((2^223-1)*2^23 + (2^22-1))*2^5 + 1)*2^3
	// + (2^2-1))*2^2 + 1) = 2^256 - 2^32 - 979 = p-2.
	var result Val
	fieldSqrN(&result.n, &x223.n, 23)
	result.Mul(&x22)
	fieldSqrN(&result.n, &result.n, 5)
	result.Mul(f)
	fieldSqrN(&result.n, &result.n, 3)
	result.Mul(&x2)
	fieldSqrN(&result.n, &result.n, 2)
	result.Mul(f)
	return f.Set(&result)
}
