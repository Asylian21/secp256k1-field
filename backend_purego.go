//go:build purego || (!arm64 && !amd64)

package field

// Generic dispatch for architectures without an assembler kernel, and for
// builds that opt out with the "purego" build tag. backendName keeps its
// "generic" default; GOSECP256K1FIELD_FORCE has no effect because there is no
// alternative backend to select.

func fieldMul(r, a, b *[5]uint64) { mulGeneric(r, a, b) }

func fieldSqr(r, a *[5]uint64) { sqrGeneric(r, a) }
