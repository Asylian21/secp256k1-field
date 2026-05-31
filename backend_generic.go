package field

import "math/bits"

// Reduction constants for the 5x52 layout.
const (
	// genM masks a full 52-bit limb.
	genM = 0xFFFFFFFFFFFFF

	// genR = (2^256 mod p) << 4 = (2^32 + 977) << 4. The <<4 accounts for the
	// 5x52 layout carrying 260 bits, so the high half starts at 2^260 = 2^4 *
	// 2^256, which is 2^4 * (2^32 + 977) mod p.
	genR = 0x1000003D10
)

// uint128 is a little-endian 128-bit accumulator built from native 64x64->128
// multiplies. Every method compiles to a handful of MUL/ADC-style instructions
// via math/bits.
type uint128 struct {
	lo, hi uint64
}

// mul sets z = a * b.
func (z *uint128) mul(a, b uint64) {
	z.hi, z.lo = bits.Mul64(a, b)
}

// addMul sets z = z + a * b.
func (z *uint128) addMul(a, b uint64) {
	hi, lo := bits.Mul64(a, b)
	var carry uint64
	z.lo, carry = bits.Add64(z.lo, lo, 0)
	z.hi += hi + carry
}

// add64 sets z = z + a.
func (z *uint128) add64(a uint64) {
	var carry uint64
	z.lo, carry = bits.Add64(z.lo, a, 0)
	z.hi += carry
}

// rsh shifts z right by n bits, where n is 52 or 64 (the only values used here).
func (z *uint128) rsh(n uint) {
	if n == 64 {
		z.lo, z.hi = z.hi, 0
		return
	}
	z.lo = z.lo>>n | z.hi<<(64-n)
	z.hi >>= n
}

// mulGeneric computes r = a * b mod p using the libsecp256k1 field_5x52 schedule
// (25 partial products with interleaved fast reduction). a and b are loaded into
// locals up front, so r may alias either input.
//
//nolint:dupl // mul and sqr necessarily share the same reduction shape.
func mulGeneric(r, a, b *[5]uint64) {
	const M, R = uint64(genM), uint64(genR)
	a0, a1, a2, a3, a4 := a[0], a[1], a[2], a[3], a[4]
	b0, b1, b2, b3, b4 := b[0], b[1], b[2], b[3], b[4]

	var c, d uint128
	var t3, t4, tx, u0 uint64

	d.mul(a0, b3)
	d.addMul(a1, b2)
	d.addMul(a2, b1)
	d.addMul(a3, b0)
	c.mul(a4, b4)
	d.addMul(R, c.lo)
	c.rsh(64)
	t3 = d.lo & M
	d.rsh(52)

	d.addMul(a0, b4)
	d.addMul(a1, b3)
	d.addMul(a2, b2)
	d.addMul(a3, b1)
	d.addMul(a4, b0)
	d.addMul(R<<12, c.lo)
	t4 = d.lo & M
	d.rsh(52)
	tx = t4 >> 48
	t4 &= M >> 4

	c.mul(a0, b0)
	d.addMul(a1, b4)
	d.addMul(a2, b3)
	d.addMul(a3, b2)
	d.addMul(a4, b1)
	u0 = d.lo & M
	d.rsh(52)
	u0 = (u0 << 4) | tx
	c.addMul(u0, R>>4)
	r[0] = c.lo & M
	c.rsh(52)

	c.addMul(a0, b1)
	c.addMul(a1, b0)
	d.addMul(a2, b4)
	d.addMul(a3, b3)
	d.addMul(a4, b2)
	c.addMul(d.lo&M, R)
	d.rsh(52)
	r[1] = c.lo & M
	c.rsh(52)

	c.addMul(a0, b2)
	c.addMul(a1, b1)
	c.addMul(a2, b0)
	d.addMul(a3, b4)
	d.addMul(a4, b3)
	c.addMul(R, d.lo)
	d.rsh(64)
	r[2] = c.lo & M
	c.rsh(52)

	c.addMul(R<<12, d.lo)
	c.add64(t3)
	r[3] = c.lo & M
	c.rsh(52)
	r[4] = c.lo + t4
}

// sqrGeneric computes r = a^2 mod p using the libsecp256k1 field_5x52 squaring
// schedule, which halves the number of partial products via symmetry.
//
//nolint:dupl // mul and sqr necessarily share the same reduction shape.
func sqrGeneric(r, a *[5]uint64) {
	const M, R = uint64(genM), uint64(genR)
	a0, a1, a2, a3, a4 := a[0], a[1], a[2], a[3], a[4]

	var c, d uint128
	var t3, t4, tx, u0 uint64

	d.mul(a0*2, a3)
	d.addMul(a1*2, a2)
	c.mul(a4, a4)
	d.addMul(R, c.lo)
	c.rsh(64)
	t3 = d.lo & M
	d.rsh(52)

	a4 *= 2
	d.addMul(a0, a4)
	d.addMul(a1*2, a3)
	d.addMul(a2, a2)
	d.addMul(R<<12, c.lo)
	t4 = d.lo & M
	d.rsh(52)
	tx = t4 >> 48
	t4 &= M >> 4

	c.mul(a0, a0)
	d.addMul(a1, a4)
	d.addMul(a2*2, a3)
	u0 = d.lo & M
	d.rsh(52)
	u0 = (u0 << 4) | tx
	c.addMul(u0, R>>4)
	r[0] = c.lo & M
	c.rsh(52)

	a0 *= 2
	c.addMul(a0, a1)
	d.addMul(a2, a4)
	d.addMul(a3, a3)
	c.addMul(d.lo&M, R)
	d.rsh(52)
	r[1] = c.lo & M
	c.rsh(52)

	c.addMul(a0, a2)
	c.addMul(a1, a1)
	d.addMul(a3, a4)
	c.addMul(R, d.lo)
	d.rsh(64)
	r[2] = c.lo & M
	c.rsh(52)

	c.addMul(R<<12, d.lo)
	c.add64(t3)
	r[3] = c.lo & M
	c.rsh(52)
	r[4] = c.lo + t4
}
