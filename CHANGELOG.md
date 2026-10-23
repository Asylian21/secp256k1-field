# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- ARM64 squaring uses immediate limb masks and retains `R<<12`/`R>>4` in
  registers. The reduction schedule and aliasing behavior remain identical to
  the generic backend. Ten paired Apple M5 Pro samples with Go 1.27.1 show
  4.75% less time (p<0.001); multiply and inverse show no significant change.
- Correct the magnitude specification: non-normalized limbs may reach twice
  the magnitude times the limb mask, including the separate 48-bit top limb.

### Added

- Deterministic ARM64 multiplication, squaring, and repeated-square tests at
  the exact limb limits for magnitudes 1..8, including aliases and the dcrd
  oracle.
- Paired benchmark results and rejected multiply/inverse experiments in
  `PERFORMANCE.md`. Optimizations are selected per operation; fewer
  instructions alone do not establish a throughput improvement.

## [0.2.0] - 2026-10-05

### Changed

- ARM64 multiply and square kernels use `EXTR` for wide shifts and `LDP` for
  adjacent input limbs. The generic reduction schedule and exact limb output
  stay the same.
- `Inverse` uses a fixed addition chain (255 squarings and 15 multiplications).
  The ARM64 repeated-square kernel keeps intermediate limbs in registers and
  writes only the final result.
- `SetBytes` and `PutBytesUnchecked` pack field elements with native 64-bit
  big-endian loads and stores.
- `Normalize` skips the second carry pass when its final reduction factor is
  zero. This remains a variable-time implementation for public values.

### Added

- Benchmarks, plus local Apple M5 Pro results in `PERFORMANCE.md`. On that
  machine the medians improved by 1.09× (`Mul`), 1.14× (`Square`), 1.38×
  (`Inverse`), 2.24× (`SetBytes`), 1.94× (`PutBytesUnchecked`), and 1.28×
  (`Normalize`).

## [0.1.0] - 2026-05-31

Initial release.

### Added

- `field.Val`, a secp256k1 base-field element in a 5×52-bit limb layout, with a
  method set and magnitude/normalization contract mirroring
  `decred/dcrd`'s `secp256k1.FieldVal`: `Set`/`SetInt`/`SetBytes`/`SetByteSlice`,
  `Bytes`/`PutBytes`/`PutBytesUnchecked`, `Add`/`Add2`/`AddInt`/`MulInt`,
  `Negate`/`NegateVal`, `Mul`/`Mul2`/`Square`/`SquareVal`, `Inverse`,
  `Normalize`, `Equals`, and the `IsZero`/`IsOne`/`IsOdd` predicate families.
- Portable pure-Go multiply/square backend (`math/bits`), present on every
  architecture and used as the correctness oracle for the assembler kernels.
- arm64 assembler backend (MUL/UMULH).
- amd64 assembler backend (BMI2 MULX), gated on CPU feature detection with a
  generic fallback.
- Runtime backend dispatch with the `Backend()` reporter and the
  `GOSECP256K1FIELD_FORCE=generic|asm` override.
- Test suite: differential fuzzing against dcrd, bit-for-bit asm-vs-generic
  kernel checks (including output/input aliasing), boundary vectors,
  zero-allocation checks, `-race`, and `FuzzMul`/`FuzzSquare`/`FuzzInverse`/
  `FuzzAddNegate` targets.
- Documentation: `README`, `SPEC`, `PERFORMANCE`, `CONTRIBUTING`.

[Unreleased]: https://github.com/Asylian21/secp256k1-field/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Asylian21/secp256k1-field/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Asylian21/secp256k1-field/releases/tag/v0.1.0
