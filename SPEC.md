# SPEC — representation, magnitude, and the dcrd contract

This document is the precise contract `field.Val` honors. It is what makes the
type a mechanical drop-in for `dcrd`'s `secp256k1.FieldVal` and what the
differential tests assert.

## Field

```
p = 2^256 - 2^32 - 977
```

A field element is an integer in `[0, p)` once normalized.

## Internal representation (5×52)

A `Val` holds five `uint64` limbs in base `2^52` (the libsecp256k1 `field_5x52`
layout), little-endian by limb:

```
value = n[0] + n[1]*2^52 + n[2]*2^104 + n[3]*2^156 + n[4]*2^208
```

Limb 4 is the top 48 bits (52·4 + 48 = 256). Each limb has spare high bits so
that several additions/negations can accumulate before a carry-propagating
normalization is required.

Reduction exploits the special form of `p`: `2^256 ≡ 2^32 + 977 (mod p)`. In the
5×52 layout the high half begins at `2^260 = 2^4·2^256`, so the fold constant is
`R = (2^32 + 977) << 4 = 0x1000003D10`.

## Normalization

A value is **normalized** when every limb is within its base and the represented
integer is the unique residue in `[0, p)`. The following require a normalized
input (identical to dcrd):

- `Equals`, `IsZero`/`IsZeroBit`, `IsOne`/`IsOneBit`, `IsOdd`/`IsOddBit`
- `Bytes`, `PutBytes`, `PutBytesUnchecked`

Call `Normalize()` first. `Normalize` always yields magnitude 1.

## Magnitude

**Magnitude** is the maximum multiple of the limb base a limb may hold. It tracks
how far a value has drifted from normalized form and bounds the size of partial
products so multiply/square cannot overflow the 128-bit accumulator.

| Operation | Input precondition | Output magnitude |
| --- | --- | --- |
| `Normalize` | any | 1 |
| `SetInt`, `SetBytes`, `Set` | — | 1 |
| `Add` / `Add2` | — | sum of input magnitudes |
| `AddInt` | — | +1 |
| `MulInt(k)` | mag·k ≤ 32 | mag·k |
| `Negate(m)` / `NegateVal(v, m)` | input mag ≤ m ≤ 31 | m+1 |
| `Mul` / `Mul2` / `Square` / `SquareVal` | each input mag ≤ 8 | 1 |
| `Inverse` | input mag ≤ 8 | 1 |

There are **no runtime checks** for these preconditions — they are the caller's
responsibility, exactly as in dcrd. Violating them (e.g. multiplying a magnitude
> 8 value) can overflow and produce a wrong result.

### Why magnitude 8 is safe for multiply

A limb at magnitude `m` is `< m·2^52`. The largest partial-product column sums a
handful of `a_i·b_j` terms; at `m ≤ 8` each operand limb is `< 8·2^52 < 2^55`, so
every 64×64→128 product is `< 2^110` and the column sums stay within the 128-bit
accumulator with room for the reduction fold. This matches dcrd's documented
limit.

## Aliasing

The result may alias either input. `r.Mul2(a, b)` is valid when `r == a`,
`r == b`, or `r == a == b`; `r.SquareVal(a)` is valid when `r == a`. (`Mul` and
`Square` are the in-place forms and rely on this.) All three backends — generic,
arm64, amd64 — honor aliasing, and the asm-vs-generic test exercises it directly.

## Canonical bytes (the bridge)

`SetBytes(*[32]byte) uint32` loads a 32-byte big-endian value and returns 1 if it
was ≥ p (i.e. reduced), else 0. `Bytes` / `PutBytes` / `PutBytesUnchecked` emit
the 32-byte big-endian encoding of the normalized value. This canonical encoding
is the interop boundary with dcrd or any other implementation.

## Correctness definition

Correctness is defined as **bit-identical normalized 32-byte output** versus
dcrd for the same inputs. Denormalized intermediates need not match dcrd
limb-for-limb; only the observable canonical result does. Separately, each
assembler kernel must match the generic backend **limb-for-limb** (a stronger
property the tests also assert).
