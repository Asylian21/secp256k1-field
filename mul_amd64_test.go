//go:build amd64 && !purego

package field

import (
	"os"
	"testing"

	"golang.org/x/sys/cpu"
)

// TestAsmMatchesGenericAMD64 pins the amd64 kernel to the portable backend
// bit-for-bit. It is skipped on CPUs without BMI2, where MULX is unavailable
// and the kernel must not be executed. Set SECP_TEST_FORCE_AMD64_ASM=1 to run
// it anyway (used to probe whether Rosetta 2 supports MULX on Apple Silicon).
func TestAsmMatchesGenericAMD64(t *testing.T) {
	if !cpu.X86.HasBMI2 && os.Getenv("SECP_TEST_FORCE_AMD64_ASM") != "1" {
		t.Skip("CPU lacks BMI2 (MULX); amd64 assembler kernel not runnable")
	}
	runAsmVsGeneric(t, mulAmd64, sqrAmd64)
}
