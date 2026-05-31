//go:build arm64 && !purego

package field

import "os"

// mulArm64 and sqrArm64 are implemented in mul_arm64.s using 64x64->128 MUL /
// UMULH and the libsecp256k1 field_5x52 reduction schedule.
//
//go:noescape
func mulArm64(r, a, b *[5]uint64)

//go:noescape
func sqrArm64(r, a *[5]uint64)

// useASM is fixed at init and only read afterwards.
var useASM = true

func init() {
	// arm64 always provides the assembler kernel; the only choice is the
	// GOSECP256K1FIELD_FORCE override.
	backendName = "arm64"
	switch os.Getenv(forceEnv) {
	case "", "asm":
		// Use the assembler backend (the default).
	case "generic":
		useASM = false
		backendName = "generic"
	default:
		// Unrecognized value: fall back to the portable backend rather than
		// failing.
		useASM = false
		backendName = "generic"
	}
}

func fieldMul(r, a, b *[5]uint64) {
	if useASM {
		mulArm64(r, a, b)
	} else {
		mulGeneric(r, a, b)
	}
}

func fieldSqr(r, a *[5]uint64) {
	if useASM {
		sqrArm64(r, a)
	} else {
		sqrGeneric(r, a)
	}
}
