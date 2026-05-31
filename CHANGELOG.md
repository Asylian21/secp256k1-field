# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Asylian21/secp256k1-field/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Asylian21/secp256k1-field/releases/tag/v0.1.0
