//go:build arm64 && !purego

#include "textflag.h"

// Assembler implementation of the secp256k1 base-field multiply and square in
// the 5x52 limb layout. This is an instruction-level translation of the
// portable mulGeneric/sqrGeneric (the libsecp256k1 field_5x52 schedule): every
// 64x64->128 partial product uses MUL (low) + UMULH (high), and the two 128-bit
// accumulators c and d live in register pairs. Correctness is guaranteed by the
// differential tests, which compare this kernel bit-for-bit against the generic
// backend and the dcrd oracle.

#define RP   R0          // result pointer
#define A0   R1
#define A1   R2
#define A2   R3
#define A3   R4
#define A4   R5
#define B0   R6
#define B1   R7
#define B2   R8
#define B3   R9
#define B4   R10
#define CLO  R11         // accumulator c, low 64
#define CHI  R12         // accumulator c, high 64
#define DLO  R13         // accumulator d, low 64
#define DHI  R14         // accumulator d, high 64
#define T3   R15
#define T4   R16
#define TX   R17
#define U0   R19
#define MM   R20         // 0xFFFFFFFFFFFFF  (2^52 - 1)
#define RR   R21         // 0x1000003D10     ((2^256 mod p) << 4)
#define PLO  R22         // partial product, low 64
#define PHI  R23         // partial product, high 64
#define S0   R24         // scratch
#define S1   R25         // scratch (pointer staging)

// func mulArm64(r, a, b *[5]uint64)
TEXT ·mulArm64(SB), NOSPLIT|NOFRAME, $0-24
	MOVD r+0(FP), RP
	MOVD a+8(FP), S0
	MOVD b+16(FP), S1

	MOVD 0(S0), A0
	MOVD 8(S0), A1
	MOVD 16(S0), A2
	MOVD 24(S0), A3
	MOVD 32(S0), A4
	MOVD 0(S1), B0
	MOVD 8(S1), B1
	MOVD 16(S1), B2
	MOVD 24(S1), B3
	MOVD 32(S1), B4

	MOVD $0xFFFFFFFFFFFFF, MM
	MOVD $0x1000003D10, RR

	// d = a0*b3 + a1*b2 + a2*b1 + a3*b0
	MUL   A0, B3, DLO
	UMULH A0, B3, DHI
	MUL   A1, B2, PLO
	UMULH A1, B2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A2, B1, PLO
	UMULH A2, B1, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A3, B0, PLO
	UMULH A3, B0, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c = a4*b4
	MUL   A4, B4, CLO
	UMULH A4, B4, CHI

	// d += R * c.lo ; c >>= 64
	MUL   RR, CLO, PLO
	UMULH RR, CLO, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MOVD  CHI, CLO
	MOVD  ZR, CHI

	// t3 = d.lo & M ; d >>= 52
	AND   MM, DLO, T3
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// d += a0*b4 + a1*b3 + a2*b2 + a3*b1 + a4*b0
	MUL   A0, B4, PLO
	UMULH A0, B4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A1, B3, PLO
	UMULH A1, B3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A2, B2, PLO
	UMULH A2, B2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A3, B1, PLO
	UMULH A3, B1, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A4, B0, PLO
	UMULH A4, B0, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// d += (R<<12) * c.lo
	LSL   $12, RR, S0
	MUL   S0, CLO, PLO
	UMULH S0, CLO, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// t4 = d.lo & M ; d >>= 52
	AND   MM, DLO, T4
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// tx = t4 >> 48 ; t4 &= M>>4
	LSR   $48, T4, TX
	LSR   $4, MM, S0
	AND   S0, T4, T4

	// c = a0*b0
	MUL   A0, B0, CLO
	UMULH A0, B0, CHI

	// d += a1*b4 + a2*b3 + a3*b2 + a4*b1
	MUL   A1, B4, PLO
	UMULH A1, B4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A2, B3, PLO
	UMULH A2, B3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A3, B2, PLO
	UMULH A3, B2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A4, B1, PLO
	UMULH A4, B1, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// u0 = d.lo & M ; d >>= 52 ; u0 = (u0<<4)|tx
	AND   MM, DLO, U0
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI
	LSL   $4, U0, U0
	ORR   TX, U0, U0

	// c += u0 * (R>>4)
	LSR   $4, RR, S0
	MUL   U0, S0, PLO
	UMULH U0, S0, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// r[0] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 0(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// c += a0*b1 + a1*b0
	MUL   A0, B1, PLO
	UMULH A0, B1, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MUL   A1, B0, PLO
	UMULH A1, B0, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// d += a2*b4 + a3*b3 + a4*b2
	MUL   A2, B4, PLO
	UMULH A2, B4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A3, B3, PLO
	UMULH A3, B3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A4, B2, PLO
	UMULH A4, B2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c += (d.lo & M) * R ; d >>= 52
	AND   MM, DLO, S0
	MUL   S0, RR, PLO
	UMULH S0, RR, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// r[1] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 8(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// c += a0*b2 + a1*b1 + a2*b0
	MUL   A0, B2, PLO
	UMULH A0, B2, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MUL   A1, B1, PLO
	UMULH A1, B1, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MUL   A2, B0, PLO
	UMULH A2, B0, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// d += a3*b4 + a4*b3
	MUL   A3, B4, PLO
	UMULH A3, B4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A4, B3, PLO
	UMULH A4, B3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c += R * d.lo ; d >>= 64
	MUL   RR, DLO, PLO
	UMULH RR, DLO, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MOVD  DHI, DLO
	MOVD  ZR, DHI

	// r[2] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 16(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// c += (R<<12) * d.lo
	LSL   $12, RR, S0
	MUL   S0, DLO, PLO
	UMULH S0, DLO, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// c += t3
	ADDS  T3, CLO, CLO
	ADC   ZR, CHI, CHI

	// r[3] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 24(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// r[4] = c.lo + t4
	ADD   T4, CLO, S0
	MOVD  S0, 32(RP)

	RET

// func sqrArm64(r, a *[5]uint64)
TEXT ·sqrArm64(SB), NOSPLIT|NOFRAME, $0-16
	MOVD r+0(FP), RP
	MOVD a+8(FP), S0

	MOVD 0(S0), A0
	MOVD 8(S0), A1
	MOVD 16(S0), A2
	MOVD 24(S0), A3
	MOVD 32(S0), A4

	MOVD $0xFFFFFFFFFFFFF, MM
	MOVD $0x1000003D10, RR

	// d = (a0*2)*a3 + (a1*2)*a2
	LSL   $1, A0, S0
	MUL   S0, A3, DLO
	UMULH S0, A3, DHI
	LSL   $1, A1, S0
	MUL   S0, A2, PLO
	UMULH S0, A2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c = a4*a4
	MUL   A4, A4, CLO
	UMULH A4, A4, CHI

	// d += R * c.lo ; c >>= 64
	MUL   RR, CLO, PLO
	UMULH RR, CLO, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MOVD  CHI, CLO
	MOVD  ZR, CHI

	// t3 = d.lo & M ; d >>= 52
	AND   MM, DLO, T3
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// a4 *= 2
	LSL   $1, A4, A4

	// d += a0*a4 + (a1*2)*a3 + a2*a2
	MUL   A0, A4, PLO
	UMULH A0, A4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	LSL   $1, A1, S0
	MUL   S0, A3, PLO
	UMULH S0, A3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A2, A2, PLO
	UMULH A2, A2, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// d += (R<<12) * c.lo
	LSL   $12, RR, S0
	MUL   S0, CLO, PLO
	UMULH S0, CLO, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// t4 = d.lo & M ; d >>= 52
	AND   MM, DLO, T4
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// tx = t4 >> 48 ; t4 &= M>>4
	LSR   $48, T4, TX
	LSR   $4, MM, S0
	AND   S0, T4, T4

	// c = a0*a0
	MUL   A0, A0, CLO
	UMULH A0, A0, CHI

	// d += a1*a4 + (a2*2)*a3
	MUL   A1, A4, PLO
	UMULH A1, A4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	LSL   $1, A2, S0
	MUL   S0, A3, PLO
	UMULH S0, A3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// u0 = d.lo & M ; d >>= 52 ; u0 = (u0<<4)|tx
	AND   MM, DLO, U0
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI
	LSL   $4, U0, U0
	ORR   TX, U0, U0

	// c += u0 * (R>>4)
	LSR   $4, RR, S0
	MUL   U0, S0, PLO
	UMULH U0, S0, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// r[0] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 0(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// a0 *= 2
	LSL   $1, A0, A0

	// c += a0*a1
	MUL   A0, A1, PLO
	UMULH A0, A1, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// d += a2*a4 + a3*a3
	MUL   A2, A4, PLO
	UMULH A2, A4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI
	MUL   A3, A3, PLO
	UMULH A3, A3, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c += (d.lo & M) * R ; d >>= 52
	AND   MM, DLO, S0
	MUL   S0, RR, PLO
	UMULH S0, RR, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	LSR   $52, DLO, DLO
	ORR   DHI<<12, DLO, DLO
	LSR   $52, DHI, DHI

	// r[1] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 8(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// c += a0*a2 + a1*a1
	MUL   A0, A2, PLO
	UMULH A0, A2, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MUL   A1, A1, PLO
	UMULH A1, A1, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// d += a3*a4
	MUL   A3, A4, PLO
	UMULH A3, A4, PHI
	ADDS  PLO, DLO, DLO
	ADC   PHI, DHI, DHI

	// c += R * d.lo ; d >>= 64
	MUL   RR, DLO, PLO
	UMULH RR, DLO, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI
	MOVD  DHI, DLO
	MOVD  ZR, DHI

	// r[2] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 16(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// c += (R<<12) * d.lo
	LSL   $12, RR, S0
	MUL   S0, DLO, PLO
	UMULH S0, DLO, PHI
	ADDS  PLO, CLO, CLO
	ADC   PHI, CHI, CHI

	// c += t3
	ADDS  T3, CLO, CLO
	ADC   ZR, CHI, CHI

	// r[3] = c.lo & M ; c >>= 52
	AND   MM, CLO, S0
	MOVD  S0, 24(RP)
	LSR   $52, CLO, CLO
	ORR   CHI<<12, CLO, CLO
	LSR   $52, CHI, CHI

	// r[4] = c.lo + t4
	ADD   T4, CLO, S0
	MOVD  S0, 32(RP)

	RET
