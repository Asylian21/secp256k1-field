# Contributing

Thanks for your interest in `secp256k1-field`. The bar for this package is
**correctness first** — it is a cryptographic field primitive, and `dcrd`'s
`FieldVal` is the oracle every change is measured against.

## Ground rules

- **No cgo.** Pure Go plus Go assembler (`.s`) only.
- **dcrd is the oracle.** Any arithmetic change must remain bit-identical to dcrd
  on normalized output. New backends must additionally match the generic backend
  **limb-for-limb**.
- **The generic backend is the source of truth.** Assembler kernels are
  instruction-level translations of `mulGeneric`/`sqrGeneric`. Keep them in sync;
  if you change the schedule, change all backends together.
- **Honor the magnitude contract** in [SPEC.md](SPEC.md). It is unchecked at
  runtime, so tests must cover the boundaries.

## Before you open a PR

Run the full suite locally. On macOS add `-ldflags=-linkmode=external` if the
test binary fails to launch.

```sh
gofmt -l .                 # must print nothing
go vet ./...
go test ./...                                  # default backend
GOSECP256K1FIELD_FORCE=generic go test -count=1 ./...
GOSECP256K1FIELD_FORCE=asm     go test -count=1 ./...
go test -race ./...
go test -run '^$' -bench '^BenchmarkMul$' -benchmem .   # sanity-check throughput
```

If you touched assembler, also verify the kernel on the other architecture. On
Apple Silicon you can exercise the amd64 MULX kernel under Rosetta 2:

```sh
GOARCH=amd64 go test -c -o /tmp/field_amd64.test .
SECP_TEST_FORCE_AMD64_ASM=1 arch -x86_64 /tmp/field_amd64.test -test.run TestAsmMatchesGenericAMD64 -test.v
```

## Testing layers

| File | What it guards |
| --- | --- |
| `diff_test.go` | differential equality vs dcrd for every op, magnitudes 1..8 |
| `asm_common_test.go` + `mul_{arm64,amd64}_test.go` | asm kernel == generic, bit-for-bit, incl. aliasing |
| `field_test.go` | boundary vectors, predicates, zero-alloc, backend smoke |
| `fuzz_test.go` | `FuzzMul`/`FuzzSquare`/`FuzzInverse`/`FuzzAddNegate` vs dcrd |
| `bench_test.go` | micro-benchmarks vs dcrd |

A change that cannot be shown bit-identical to dcrd will not be merged.

## Style

- Comments explain **intent and constraints** (magnitude limits, carry bounds,
  why an instruction is/ isn't used), not what each line does.
- Keep the assembler readable: one commented block per partial-product /
  reduction step, mirroring the generic backend's structure.
