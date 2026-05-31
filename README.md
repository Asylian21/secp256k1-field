# secp256k1-field

**Fast secp256k1 finite-field arithmetic for Go, tuned for Bitcoin and ECDSA
research workloads.**

[![CI](https://github.com/Asylian21/secp256k1-field/actions/workflows/ci.yml/badge.svg)](https://github.com/Asylian21/secp256k1-field/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Asylian21/secp256k1-field.svg)](https://pkg.go.dev/github.com/Asylian21/secp256k1-field)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`secp256k1-field` is a high-performance Go library for fixed-precision
arithmetic over the secp256k1 base field:

```text
p = 2^256 - 2^32 - 977
```

It provides `field.Val`, a high-throughput field element for Go developers
building Bitcoin research tools, ECDSA / secp256k1 benchmarks, affine point
walking experiments, public key-space simulations, or performance-sensitive
cryptography tooling. The public method set and magnitude semantics mirror
[`decred/dcrd`'s `secp256k1.FieldVal`](https://pkg.go.dev/github.com/decred/dcrd/dcrec/secp256k1/v4),
so existing hot loops can usually switch with small, mechanical edits.

Internally, the package uses the libsecp256k1-style **5x52-bit limb layout**,
mapping field multiplication to native `64x64->128` products instead of a
10x26 schoolbook schedule.

## Who This Is For

Use this package when you need fast, allocation-free arithmetic modulo the
secp256k1 prime in Go and your workload is built around public or non-secret
field values:

- Bitcoin, ECDSA, and secp256k1 researchers comparing field implementations.
- Go developers porting tight loops from dcrd's `secp256k1.FieldVal`.
- Benchmark authors measuring 5x52-bit limbs, BMI2 `MULX`, arm64 `UMULH`, or
  generic `math/bits` arithmetic.
- Engineers building point-walking, simulation, fuzzing, or educational tools
  where throughput matters more than constant-time behavior.

If you are processing private keys, wallet seed material, signing secrets, or
any value where timing side channels matter, use a constant-time secp256k1
library instead.

## Highlights

- **Pure Go by default, no cgo.** The portable backend is built on `math/bits`,
  works on every Go-supported architecture, and acts as the correctness reference.
- **Hand-written assembler where it matters.** `arm64` uses `MUL`/`UMULH`;
  `amd64` uses BMI2 `MULX` when available, with automatic generic fallback.
- **FieldVal-compatible API.** Method names, chaining style, normalization rules, and
  magnitude limits follow dcrd's `FieldVal` contract.
- **Zero allocations on the hot path.** Multiplication, squaring, normalization,
  and serialization are designed for tight loops.
- **Cross-checked correctness.** Normalized outputs are differential-tested
  against dcrd, while assembler kernels are compared limb-for-limb against the
  generic backend.

## Security Model

This package is optimized for **non-secret, throughput-oriented workloads** such
as research, simulation, benchmarking, and public key-space experiments.

Operations are **not guaranteed to be constant time**. Do not use this package to
process secret keys, private scalars, wallet seed material, or any value where a
timing side channel would matter.

## Installation

```sh
go get github.com/Asylian21/secp256k1-field
```

Requires Go 1.22 or newer.

## Quick Start

```go
package main

import (
	"fmt"

	field "github.com/Asylian21/secp256k1-field"
)

func main() {
	var a, b, r field.Val
	a.SetInt(7)
	b.SetInt(6)

	r.Mul2(&a, &b).Normalize() // r = a*b mod p

	fmt.Println(r.String())
	fmt.Println("backend:", field.Backend()) // generic | arm64 | amd64
}
```

## API Overview

`field.Val` follows the dcrd `FieldVal` surface used by point-walking code:

| Category | Methods |
| --- | --- |
| Load | `Set`, `SetInt`, `SetBytes`, `SetByteSlice`, `Zero` |
| Store | `Bytes`, `PutBytes`, `PutBytesUnchecked`, `String` |
| Add / negate | `Add`, `Add2`, `AddInt`, `Negate`, `NegateVal`, `MulInt` |
| Multiply | `Mul`, `Mul2`, `Square`, `SquareVal` |
| Field operations | `Normalize`, `Inverse`, `Equals` |
| Predicates | `IsZero`, `IsZeroBit`, `IsOne`, `IsOneBit`, `IsOdd`, `IsOddBit` |
| Backend | `Backend() string` |

The important contract is the same as dcrd: callers must respect normalization
and magnitude preconditions. Values should be normalized before comparison,
oddness checks, zero checks, or serialization. Inputs to multiply, square, and
inverse must have magnitude at most 8.

See [SPEC.md](SPEC.md) for the exact representation, magnitude rules, aliasing
guarantees, and canonical byte encoding.

## Backend Selection

Multiplication and squaring dispatch once at package initialization:

- `arm64`: assembler backend by default.
- `amd64`: assembler backend when BMI2 is available; generic otherwise.
- Other architectures: portable pure-Go backend.

`field.Backend()` reports the selected implementation. For testing or benchmark
control, set:

```sh
GOSECP256K1FIELD_FORCE=generic   # force portable backend
GOSECP256K1FIELD_FORCE=asm       # request assembler backend when available
```

Unsupported or unknown values fall back to the generic backend.

## Performance

Apple M3, `go test -bench`, steady state, compared with dcrd's pure-Go
10x26 `FieldVal`. All listed operations report `0 allocs/op`.

| Operation | dcrd | generic | arm64 asm | asm speedup |
| --- | ---: | ---: | ---: | ---: |
| `Mul` | 38.8 ns | 21.3 ns | **11.8 ns** | **3.28x** |
| `Square` | 25.3 ns | 14.6 ns | **8.85 ns** | **2.86x** |
| `Inverse` | 11.24 microseconds | - | **4.00 microseconds** | **2.81x** |

See [PERFORMANCE.md](PERFORMANCE.md) for methodology, benchmark commands, and
notes on the assembler design.

## Correctness

The test suite treats dcrd's `FieldVal` as the external oracle and the generic
backend as the internal oracle for assembler kernels.

It covers:

- Differential tests for multiplication, squaring, addition, negation, inverse,
  normalization, serialization, equality, oddness, and boundary values.
- Forced-backend runs for `generic` and `asm`.
- Limb-for-limb assembler checks against the generic backend, including output
  aliasing.
- Boundary vectors for `0`, `1`, `p-1`, values near `p`, overflow, and
  `2^256-1`.
- Allocation checks, race testing, and fuzz targets.

Run the core suite:

```sh
go test ./...
GOSECP256K1FIELD_FORCE=generic go test -count=1 ./...
GOSECP256K1FIELD_FORCE=asm     go test -count=1 ./...
go test -race ./...
```

## Documentation

- [SPEC.md](SPEC.md) defines the representation, normalization, magnitude, and
  interoperability contract.
- [PERFORMANCE.md](PERFORMANCE.md) explains the benchmark methodology and
  assembler design.
- [CONTRIBUTING.md](CONTRIBUTING.md) describes the standards for tests, style,
  and backend changes.
- [CHANGELOG.md](CHANGELOG.md) tracks releases using Semantic Versioning.

## Search Keywords

`secp256k1`, `secp256k1 field arithmetic`, `Go secp256k1`, `Bitcoin
cryptography`, `ECDSA field arithmetic`, `finite field arithmetic`, `dcrd
FieldVal`, `libsecp256k1 field_5x52`, `5x52 limbs`, `BMI2 MULX`, `arm64 UMULH`,
`zero allocation Go crypto benchmarks`.

Suggested GitHub repository description:

```text
Fast secp256k1 finite-field arithmetic for Go: dcrd FieldVal-compatible API, 5x52 limbs, pure Go plus arm64/amd64 assembly.
```

Suggested GitHub topics:

```text
secp256k1 bitcoin ecdsa cryptography finite-field go golang dcrd libsecp256k1 field-arithmetic elliptic-curves benchmarking assembly arm64 amd64 bmi2
```

## Open Source

This project is open source under the [MIT License](LICENSE). You are free to
use, copy, modify, merge, publish, distribute, sublicense, and sell copies of the
software under the terms of the license.

## Support This Project ₿

If this project helped you understand Bitcoin security, benchmark Go code, or
explain why brute force is not a business model, you can support continued
research here:

Bitcoin donation address:

```text
bc1q9c5mmx9d3ajevjrvvw9yf52jclsre8x86qhnak
```

Every satoshi helps fund more experiments, better documentation, and fewer
hand-wavy claims about cryptography.

## License

[MIT](LICENSE). Copyright (c) 2026 David Zita.
