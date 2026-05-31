// Package field implements fast fixed-precision arithmetic over the secp256k1
// base field Fp, where p = 2^256 - 2^32 - 977.
//
// It is a drop-in-shaped replacement for the field type used by elliptic-curve
// code that walks many points (for example batched affine point addition with a
// single Montgomery inversion): the method set and magnitude semantics mirror
// the decred/dcrd secp256k1 FieldVal so existing hot loops can switch types with
// minimal changes, while the internals use a 5x52-bit limb layout that maps
// directly onto native 64x64->128 multiplies.
//
// # Representation
//
// A field element is stored as five uint64 limbs in base 2^52 (the libsecp256k1
// field_5x52 layout). Multiplication and squaring therefore perform 25 wide
// 64x64->128 products instead of the 100 narrow 32x32->64 products required by a
// 10x26 schoolbook, with far less carry bookkeeping. The reduction exploits the
// special form of p (2^256 = 2^32 + 977 mod p) for a cheap fold of the high
// limbs.
//
// # Normalization and magnitude
//
// As with the dcrd FieldVal, the representation keeps spare bits per limb so a
// run of additions and negations can be performed without propagating carries.
// Two concepts must be tracked by the caller:
//
//   - Normalization: comparisons ([Val.Equals]), oddness ([Val.IsOddBit]),
//     zeroness ([Val.IsZero]) and serialization ([Val.Bytes],
//     [Val.PutBytesUnchecked]) require a normalized input. Call [Val.Normalize]
//     first.
//   - Magnitude: the maximum multiple of the limb base that a limb may hold. A
//     normalized value or a multiply/square result has magnitude 1. [Val.Add] /
//     [Val.Add2] add magnitudes; [Val.NegateVal] raises magnitude by one. Inputs
//     to [Val.Mul] / [Val.Mul2] / [Val.Square] / [Val.SquareVal] / [Val.Inverse]
//     must have magnitude at most 8. There are no runtime checks for these
//     preconditions; they are the caller's responsibility, exactly as in dcrd.
//
// # Backend selection
//
// Multiply and square dispatch once at package initialization to the fastest
// backend available for the current architecture. A portable pure-Go backend
// (built on math/bits) is always present and also serves as the correctness
// oracle for the assembler backends. On arm64 and amd64 a hand-written assembler
// backend is used by default (on amd64 only when the CPU advertises BMI2, which
// provides the flag-free MULX multiply; otherwise it falls back to the generic
// backend). [Backend] reports the kernel that actually runs.
//
// The environment variable GOSECP256K1FIELD_FORCE may be set to "generic" to pin
// the pure-Go backend, or "asm" to request the assembler backend when one is
// implemented for the current architecture. An unknown value, or "asm" on a
// build/CPU without an assembler backend, falls back to generic rather than
// failing.
//
// # Security
//
// This package targets throughput for non-secret workloads such as key-space
// research and benchmarking. Operations are NOT guaranteed to run in constant
// time. Do not use it to process secret keys where timing side channels matter.
package field
