//go:build arm64 && !purego

package field

import "testing"

// TestAsmMatchesGenericARM64 pins the arm64 kernel to the portable backend
// bit-for-bit, independent of which backend the package selected at init.
func TestAsmMatchesGenericARM64(t *testing.T) {
	runAsmVsGeneric(t, mulArm64, sqrArm64)
}
