//go:build amd64 && !purego

#include "textflag.h"

// Assembler implementation of the secp256k1 base-field multiply and square in
// the 5x52 limb layout, for CPUs with BMI2 (MULX). It is an instruction-level
// translation of the portable mulGeneric/sqrGeneric (the libsecp256k1
// field_5x52 schedule): each 64x64->128 partial product uses MULX (which does
// not touch the flags), and the two 128-bit accumulators c and d are summed
// with a single ADD/ADC carry chain. Correctness is guaranteed by the
// differential tests, which compare this kernel bit-for-bit against the generic
// backend and the dcrd oracle.
//
// Aliasing: the result pointer r may alias a and/or b (the consumer relies on
// this via Mul/Square in place and the Inverse ladder). The schedule's last
// read of a and b happens just before r[2] is stored, so only the r[0] and r[1]
// stores would otherwise clobber an input limb. Those two limbs are therefore
// held in the already-dead registers U0 (r[0]) and TX (r[1]) and flushed to
// memory only after every input read has completed. This costs no extra memory
// traffic and keeps the kernel frame-free (so BP stays available for the mask).
//
// Register usage:
//   SI = a pointer      DI = b pointer      BX = r pointer      BP = M mask
//   R8:R9   = c (lo:hi)     R10:R11 = d (lo:hi)
//   R12 = t3   R13 = t4   R14 = tx/scratch, later deferred r[1]
//   R15 = u0/scratch, later deferred r[0]
//   DX = MULX multiplier   AX = product lo / scratch   CX = product hi

#define CLO R8
#define CHI R9
#define DLO R10
#define DHI R11
#define T3  R12
#define T4  R13
#define TX  R14
#define U0  R15
#define MM  BP

// func mulAmd64(r, a, b *[5]uint64)
TEXT ·mulAmd64(SB), NOSPLIT|NOFRAME, $0-24
	MOVQ r+0(FP), BX
	MOVQ a+8(FP), SI
	MOVQ b+16(FP), DI
	MOVQ $0xFFFFFFFFFFFFF, MM

	// d = a0*b3 + a1*b2 + a2*b1 + a3*b0
	MOVQ  0(SI), DX
	MULXQ 24(DI), DLO, DHI
	MOVQ  8(SI), DX
	MULXQ 16(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  16(SI), DX
	MULXQ 8(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  24(SI), DX
	MULXQ 0(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// c = a4*b4
	MOVQ  32(SI), DX
	MULXQ 32(DI), CLO, CHI

	// d += R * c.lo ; c >>= 64
	MOVQ  $0x1000003D10, DX
	MULXQ CLO, AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  CHI, CLO
	XORQ  CHI, CHI

	// t3 = d.lo & M ; d >>= 52
	MOVQ  DLO, T3
	ANDQ  MM, T3
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// d += a0*b4 + a1*b3 + a2*b2 + a3*b1 + a4*b0
	MOVQ  0(SI), DX
	MULXQ 32(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  8(SI), DX
	MULXQ 24(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  16(SI), DX
	MULXQ 16(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  24(SI), DX
	MULXQ 8(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  32(SI), DX
	MULXQ 0(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// d += (R<<12) * c.lo
	MOVQ  $0x1000003D10000, DX
	MULXQ CLO, AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// t4 = d.lo & M ; d >>= 52
	MOVQ  DLO, T4
	ANDQ  MM, T4
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// tx = t4 >> 48 ; t4 &= M>>4
	MOVQ  T4, TX
	SHRQ  $48, TX
	MOVQ  MM, AX
	SHRQ  $4, AX
	ANDQ  AX, T4

	// c = a0*b0
	MOVQ  0(SI), DX
	MULXQ 0(DI), CLO, CHI

	// d += a1*b4 + a2*b3 + a3*b2 + a4*b1
	MOVQ  8(SI), DX
	MULXQ 32(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  16(SI), DX
	MULXQ 24(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  24(SI), DX
	MULXQ 16(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  32(SI), DX
	MULXQ 8(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// u0 = d.lo & M ; d >>= 52 ; u0 = (u0<<4)|tx
	MOVQ  DLO, U0
	ANDQ  MM, U0
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI
	SHLQ  $4, U0
	ORQ   TX, U0

	// c += u0 * (R>>4)
	MOVQ  $0x1000003D1, DX
	MULXQ U0, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// r[0] = c.lo & M (deferred in U0 until inputs are fully read) ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, U0
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += a0*b1 + a1*b0
	MOVQ  0(SI), DX
	MULXQ 8(DI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  8(SI), DX
	MULXQ 0(DI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// d += a2*b4 + a3*b3 + a4*b2
	MOVQ  16(SI), DX
	MULXQ 32(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  24(SI), DX
	MULXQ 24(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  32(SI), DX
	MULXQ 16(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// c += (d.lo & M) * R ; d >>= 52
	MOVQ  DLO, TX
	ANDQ  MM, TX
	MOVQ  $0x1000003D10, DX
	MULXQ TX, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// r[1] = c.lo & M (deferred in TX until inputs are fully read) ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, TX
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += a0*b2 + a1*b1 + a2*b0
	MOVQ  0(SI), DX
	MULXQ 16(DI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  8(SI), DX
	MULXQ 8(DI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  16(SI), DX
	MULXQ 0(DI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// d += a3*b4 + a4*b3  (final reads of a and b)
	MOVQ  24(SI), DX
	MULXQ 32(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  32(SI), DX
	MULXQ 24(DI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// All reads of a and b are complete; flush the deferred low limbs so that
	// r is free to alias a or b.
	MOVQ  U0, 0(BX)
	MOVQ  TX, 8(BX)

	// c += R * d.lo ; d >>= 64
	MOVQ  $0x1000003D10, DX
	MULXQ DLO, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  DHI, DLO
	XORQ  DHI, DHI

	// r[2] = c.lo & M ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, 16(BX)
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += (R<<12) * d.lo
	MOVQ  $0x1000003D10000, DX
	MULXQ DLO, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// c += t3
	ADDQ  T3, CLO
	ADCQ  $0, CHI

	// r[3] = c.lo & M ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, 24(BX)
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// r[4] = c.lo + t4
	ADDQ  T4, CLO
	MOVQ  CLO, 32(BX)

	RET

// func sqrAmd64(r, a *[5]uint64)
// Squaring uses the same schedule with halved partial products. The doubled
// operand of each product is formed in DX (the MULX multiplier) via ADDQ DX,DX,
// which is why every doubled term places that operand in DX and reads the other
// operand from memory. The same r[0]/r[1] deferral as mulAmd64 makes r safe to
// alias a (used by Square in place and throughout the Inverse ladder).
TEXT ·sqrAmd64(SB), NOSPLIT|NOFRAME, $0-16
	MOVQ r+0(FP), BX
	MOVQ a+8(FP), SI
	MOVQ $0xFFFFFFFFFFFFF, MM

	// d = (2*a0)*a3 + (2*a1)*a2
	MOVQ  0(SI), DX
	ADDQ  DX, DX
	MULXQ 24(SI), DLO, DHI
	MOVQ  8(SI), DX
	ADDQ  DX, DX
	MULXQ 16(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// c = a4*a4
	MOVQ  32(SI), DX
	MULXQ 32(SI), CLO, CHI

	// d += R * c.lo ; c >>= 64
	MOVQ  $0x1000003D10, DX
	MULXQ CLO, AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  CHI, CLO
	XORQ  CHI, CHI

	// t3 = d.lo & M ; d >>= 52
	MOVQ  DLO, T3
	ANDQ  MM, T3
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// d += a0*(2*a4) + (2*a1)*a3 + a2*a2
	MOVQ  32(SI), DX
	ADDQ  DX, DX
	MULXQ 0(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  8(SI), DX
	ADDQ  DX, DX
	MULXQ 24(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  16(SI), DX
	MULXQ 16(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// d += (R<<12) * c.lo
	MOVQ  $0x1000003D10000, DX
	MULXQ CLO, AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// t4 = d.lo & M ; d >>= 52
	MOVQ  DLO, T4
	ANDQ  MM, T4
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// tx = t4 >> 48 ; t4 &= M>>4
	MOVQ  T4, TX
	SHRQ  $48, TX
	MOVQ  MM, AX
	SHRQ  $4, AX
	ANDQ  AX, T4

	// c = a0*a0
	MOVQ  0(SI), DX
	MULXQ 0(SI), CLO, CHI

	// d += a1*(2*a4) + (2*a2)*a3
	MOVQ  32(SI), DX
	ADDQ  DX, DX
	MULXQ 8(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  16(SI), DX
	ADDQ  DX, DX
	MULXQ 24(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// u0 = d.lo & M ; d >>= 52 ; u0 = (u0<<4)|tx
	MOVQ  DLO, U0
	ANDQ  MM, U0
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI
	SHLQ  $4, U0
	ORQ   TX, U0

	// c += u0 * (R>>4)
	MOVQ  $0x1000003D1, DX
	MULXQ U0, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// r[0] = c.lo & M (deferred in U0 until inputs are fully read) ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, U0
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += (2*a0)*a1
	MOVQ  0(SI), DX
	ADDQ  DX, DX
	MULXQ 8(SI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// d += a2*(2*a4) + a3*a3
	MOVQ  32(SI), DX
	ADDQ  DX, DX
	MULXQ 16(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI
	MOVQ  24(SI), DX
	MULXQ 24(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// c += (d.lo & M) * R ; d >>= 52
	MOVQ  DLO, TX
	ANDQ  MM, TX
	MOVQ  $0x1000003D10, DX
	MULXQ TX, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  DHI, AX
	SHLQ  $12, AX
	SHRQ  $52, DLO
	ORQ   AX, DLO
	SHRQ  $52, DHI

	// r[1] = c.lo & M (deferred in TX until inputs are fully read) ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, TX
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += (2*a0)*a2 + a1*a1
	MOVQ  0(SI), DX
	ADDQ  DX, DX
	MULXQ 16(SI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  8(SI), DX
	MULXQ 8(SI), AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// d += a3*(2*a4)  (final read of a)
	MOVQ  32(SI), DX
	ADDQ  DX, DX
	MULXQ 24(SI), AX, CX
	ADDQ  AX, DLO
	ADCQ  CX, DHI

	// All reads of a are complete; flush the deferred low limbs so that r is
	// free to alias a.
	MOVQ  U0, 0(BX)
	MOVQ  TX, 8(BX)

	// c += R * d.lo ; d >>= 64
	MOVQ  $0x1000003D10, DX
	MULXQ DLO, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI
	MOVQ  DHI, DLO
	XORQ  DHI, DHI

	// r[2] = c.lo & M ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, 16(BX)
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// c += (R<<12) * d.lo
	MOVQ  $0x1000003D10000, DX
	MULXQ DLO, AX, CX
	ADDQ  AX, CLO
	ADCQ  CX, CHI

	// c += t3
	ADDQ  T3, CLO
	ADCQ  $0, CHI

	// r[3] = c.lo & M ; c >>= 52
	MOVQ  CLO, AX
	ANDQ  MM, AX
	MOVQ  AX, 24(BX)
	MOVQ  CHI, AX
	SHLQ  $12, AX
	SHRQ  $52, CLO
	ORQ   AX, CLO
	SHRQ  $52, CHI

	// r[4] = c.lo + t4
	ADDQ  T4, CLO
	MOVQ  CLO, 32(BX)

	RET
