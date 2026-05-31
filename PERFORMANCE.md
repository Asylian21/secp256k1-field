# PERFORMANCE

## Why this is faster than a 10×26 schoolbook

dcrd's pure-Go `FieldVal` stores 10 limbs in base `2^26` and multiplies with 100
narrow `uint32×uint32→uint64` products plus a lot of mask/shift carry handling —
a layout chosen historically for 32-bit machines. On a 64-bit CPU that leaves the
wide multiplier idle.

`field.Val` uses 5 limbs in base `2^52`, so a multiply is **25 wide
`64×64→128` products** with far less carry bookkeeping, and reduction is a cheap
fold by `R = 0x1000003D10` using `p = 2^256 - 2^32 - 977`. The pure-Go backend
expresses the wide products with `math/bits.Mul64`/`Add64`; the assembler
backends use the CPU's native widening multiply (arm64 `MUL`/`UMULH`, amd64
`MULX`).

## Results

Apple M3 (`darwin/arm64`), `GOMAXPROCS=1` per `-cpu` default for these micro
benchmarks, steady state. Baseline is dcrd's pure-Go `FieldVal`. All operations
report **0 B/op, 0 allocs/op**.

| Operation | dcrd | generic | arm64 asm | generic× | asm× |
| --- | --- | --- | --- | --- | --- |
| `Mul`     | 38.8 ns  | 21.3 ns | **11.8 ns** | 1.82× | **3.28×** |
| `Square`  | 25.3 ns  | 14.6 ns | **8.85 ns** | 1.74× | **2.86×** |
| `Inverse` | 11.24 µs | —       | **4.00 µs** | —     | **2.81×** |

(`amd64` MULX is verified bit-for-bit against the generic backend and is wired
into CI on native BMI2 runners; throughput numbers there are reported per-run by
CI rather than pinned here.)

### Gates

The project enforces two speed gates on `Mul` versus dcrd before an
implementation is considered done:

- **generic ≥ 1.5×** — met (1.82×).
- **assembler ≥ 2.5×** — met (3.28× on arm64).

## Reproduce

```sh
# assembler (default on arm64/amd64+BMI2)
go test -run '^$' -bench '^(BenchmarkMul|BenchmarkMulDcrd|BenchmarkSquare|BenchmarkSquareDcrd|BenchmarkInverse|BenchmarkInverseDcrd)$' \
  -benchmem -count=10 .

# portable backend
GOSECP256K1FIELD_FORCE=generic go test -run '^$' -bench '^(BenchmarkMul|BenchmarkSquare)$' -benchmem -count=10 .
```

Compare runs with [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
go install golang.org/x/perf/cmd/benchstat@latest
GOSECP256K1FIELD_FORCE=generic go test -run '^$' -bench '^BenchmarkMul$' -count=10 . | tee generic.txt
go test -run '^$' -bench '^BenchmarkMul$' -count=10 . | tee asm.txt
benchstat generic.txt asm.txt
```

> macOS note: if `go test` fails to launch with a `dyld`/`LC_UUID` error, add
> `-ldflags=-linkmode=external`.

## Notes on the assembler design

- **Single carry chain.** Each kernel sums many partial products into one 128-bit
  accumulator. On amd64 this is `MULX` (flag-free multiply) plus one `ADD`/`ADC`
  chain; the ADX dual-carry instructions (`ADCX`/`ADOX`) give no benefit for this
  reduction shape, so they are intentionally not used — keeping the kernel a
  direct, verifiable mirror of the portable backend.
- **Aliasing without spills.** arm64 keeps all ten input limbs in registers, so
  the result may freely alias an input. amd64 has no spare registers, so the two
  low result limbs are held in registers that are already dead at that point and
  flushed only after the final input read — preserving aliasing safety with no
  extra memory traffic and no stack frame.
