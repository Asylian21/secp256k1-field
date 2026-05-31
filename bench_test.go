package field

import (
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// Benchmark operands: two fixed, normalized field elements that exercise full
// 256-bit limbs (no small-value shortcuts).
var (
	benchA = [32]byte{
		0x9d, 0x4f, 0x2a, 0x11, 0xc8, 0x3e, 0x7b, 0x06, 0xf1, 0x2c, 0x55, 0x90, 0xaa, 0xbe, 0x31, 0x77,
		0x42, 0x88, 0x19, 0xde, 0x6a, 0x0b, 0xcd, 0xf3, 0x10, 0x7e, 0x95, 0x24, 0x3b, 0xc6, 0x88, 0x01,
	}
	benchB = [32]byte{
		0x1a, 0xe7, 0x63, 0xbc, 0x09, 0x5f, 0x84, 0xd2, 0x71, 0x36, 0xfa, 0x4e, 0x8c, 0x02, 0x9b, 0x55,
		0xc3, 0x6d, 0x0a, 0x47, 0xb8, 0xe1, 0x29, 0x90, 0x5d, 0xf4, 0x12, 0xab, 0x73, 0x68, 0xee, 0x3c,
	}
)

// Package-level sinks defeat dead-code elimination.
var (
	sink     Val
	sinkDcrd secp.FieldVal
)

func benchOperands() (a, b Val) {
	a.SetBytes(&benchA)
	b.SetBytes(&benchB)
	a.Normalize()
	b.Normalize()
	return a, b
}

func benchOperandsDcrd() (a, b secp.FieldVal) {
	a.SetBytes(&benchA)
	b.SetBytes(&benchB)
	a.Normalize()
	b.Normalize()
	return a, b
}

func BenchmarkMul(b *testing.B) {
	x, y := benchOperands()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink.Mul2(&x, &y)
	}
}

func BenchmarkMulDcrd(b *testing.B) {
	x, y := benchOperandsDcrd()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkDcrd.Mul2(&x, &y)
	}
}

func BenchmarkSquare(b *testing.B) {
	x, _ := benchOperands()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink.SquareVal(&x)
	}
}

func BenchmarkSquareDcrd(b *testing.B) {
	x, _ := benchOperandsDcrd()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkDcrd.SquareVal(&x)
	}
}

func BenchmarkInverse(b *testing.B) {
	x, _ := benchOperands()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink.Set(&x).Inverse()
	}
}

func BenchmarkInverseDcrd(b *testing.B) {
	x, _ := benchOperandsDcrd()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkDcrd.Set(&x).Inverse()
	}
}

// BenchmarkMulSquareChain approximates the per-key field cost of a batched
// affine addition (~2 mul + 1 sqr) to make end-to-end estimates easy.
func BenchmarkMulSquareChain(b *testing.B) {
	x, y := benchOperands()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink.Mul2(&x, &y)
		sink.Mul2(&sink, &x)
		sink.SquareVal(&sink)
	}
}
