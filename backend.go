package field

// forceEnv pins the multiply/square backend. Set it to "generic" to force the
// portable backend, or "asm" to request the assembler backend on builds/CPUs
// that provide one. It is consulted once, at package initialization, by the
// architecture-specific dispatch.
const forceEnv = "GOSECP256K1FIELD_FORCE"

// backendName names the active multiply/square backend (for example "generic",
// "arm64", or "amd64"). It is assigned exactly once, during package
// initialization by the architecture-specific dispatch file, and only read
// afterwards, so concurrent reads need no synchronization.
var backendName = "generic"

// Backend returns the name of the active multiply/square backend. It reflects
// the kernel that actually runs, after CPU detection and the
// GOSECP256K1FIELD_FORCE override.
func Backend() string { return backendName }
