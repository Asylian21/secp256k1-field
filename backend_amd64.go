//go:build amd64 && !purego

package field

import (
	"os"

	"golang.org/x/sys/cpu"
)

// mulAmd64 and sqrAmd64 are implemented in mul_amd64.s using the BMI2 MULX
// instruction (a flag-free 64x64->128 multiply) with a single ADD/ADC carry
// chain. For this reduction shape - summing many partial products into a single
// 128-bit accumulator - the ADX dual-carry instructions (ADCX/ADOX) provide no
// benefit over a single carry chain, so they are intentionally not used; MULX
// alone delivers the speedup while keeping the kernel a direct, verifiable
// mirror of the portable backend.
//
//go:noescape
func mulAmd64(r, a, b *[5]uint64)

//go:noescape
func sqrAmd64(r, a *[5]uint64)

// useASM is fixed at init and only read afterwards.
var useASM bool

func init() {
	hasMULX := cpu.X86.HasBMI2

	// Default: assembler when MULX is available, generic otherwise.
	useASM = hasMULX
	if useASM {
		backendName = "amd64"
	} else {
		backendName = "generic"
	}

	switch os.Getenv(forceEnv) {
	case "":
		// Keep the default selected above.
	case "asm":
		// Honor only if the CPU can actually run the kernel.
		if hasMULX {
			useASM = true
			backendName = "amd64"
		}
	case "generic":
		useASM = false
		backendName = "generic"
	default:
		// Unrecognized value: fall back to the portable backend.
		useASM = false
		backendName = "generic"
	}
}

func fieldMul(r, a, b *[5]uint64) {
	if useASM {
		mulAmd64(r, a, b)
	} else {
		mulGeneric(r, a, b)
	}
}

func fieldSqr(r, a *[5]uint64) {
	if useASM {
		sqrAmd64(r, a)
	} else {
		sqrGeneric(r, a)
	}
}
