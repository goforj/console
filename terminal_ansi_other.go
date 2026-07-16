//go:build !windows

package console

import "io"

// terminalSupportsANSI trusts terminal detection on platforms whose terminals use ANSI control sequences natively.
func terminalSupportsANSI(int) bool {
	return true
}

// writeTerminalString writes directly because ANSI terminals need no per-write mode changes.
func writeTerminalString(writer io.Writer, value string) (int, error) {
	return io.WriteString(writer, value)
}
