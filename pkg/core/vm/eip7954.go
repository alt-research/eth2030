package vm

// EIP-7954: Increase Maximum Contract Size.
//
// Raises the deployed contract code size limit from 24,576 bytes (24 KiB) to
// 65,536 bytes (64 KiB) and the init code size limit from 49,152 bytes (48 KiB)
// to 131,072 bytes (128 KiB).

const (
	// MaxCodeSizeGlamsterdam is the EIP-7954 max deployed code size (64 KiB).
	MaxCodeSizeGlamsterdam = 65536 // 0x10000

	// MaxInitCodeSizeGlamsterdam is the EIP-7954 max init code size (128 KiB).
	MaxInitCodeSizeGlamsterdam = 131072 // 0x20000
)

// MaxCodeSizeForFork returns the maximum deployed contract code size for the
// given fork rules. Post-Glamsterdam uses EIP-7954 limits.
func MaxCodeSizeForFork(rules ForkRules) int {
	if rules.IsEIP7954 {
		return MaxCodeSizeGlamsterdam
	}
	return MaxCodeSize
}

// MaxInitCodeSizeForFork returns the maximum init code size for the given
// fork rules. Post-Glamsterdam uses EIP-7954 limits.
func MaxInitCodeSizeForFork(rules ForkRules) int {
	if rules.IsEIP7954 {
		return MaxInitCodeSizeGlamsterdam
	}
	return MaxInitCodeSize
}
