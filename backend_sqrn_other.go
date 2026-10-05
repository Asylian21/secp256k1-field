//go:build !arm64 || purego

package field

// fieldSqrN squares count times, with count > 0. Keep the selected backend on
// amd64; architectures without assembly use the same portable schedule.
func fieldSqrN(r, a *[5]uint64, count uint64) {
	fieldSqr(r, a)
	for i := uint64(1); i < count; i++ {
		fieldSqr(r, r)
	}
}
