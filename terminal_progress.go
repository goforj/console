package console

import (
	"fmt"
)

const (
	terminalProgressStateClear         = 0
	terminalProgressStateDeterminate   = 1
	terminalProgressStateIndeterminate = 3
)

// terminalProgressOwner prevents a completed concurrent display from restoring stale terminal state.
type terminalProgressOwner interface {
	transientOwner
	terminalProgressFinished() bool
}

// shouldRenderTerminalProgress keeps terminal-owned indicators out of redirected output and automation logs.
func (c *Console) shouldRenderTerminalProgress() bool {
	if c.terminalProgressEnabled == nil || !*c.terminalProgressEnabled {
		return false
	}
	descriptor, ok := writerDescriptor(c.stdout)
	if !ok || !c.isTerminal(descriptor) || !c.supportsANSI(descriptor) {
		return false
	}
	return true
}

// setTerminalProgress gives one live display ownership of the terminal's singular progress indicator.
func (c *Console) setTerminalProgress(owner terminalProgressOwner, state, progress int) {
	if !c.shouldRenderTerminalProgress() {
		return
	}
	c.terminalProgressMu.Lock()
	defer c.terminalProgressMu.Unlock()
	if owner.terminalProgressFinished() {
		return
	}
	if c.terminalProgressOwner != nil && c.terminalProgressOwner != owner {
		return
	}
	c.terminalProgressOwner = owner
	c.outputMu.Lock()
	_, _ = writeConsoleString(c.stdout, terminalProgressSequence(state, progress))
	c.outputMu.Unlock()
}

// clearTerminalProgress prevents a completed owner from disturbing a newer terminal progress display.
func (c *Console) clearTerminalProgress(owner terminalProgressOwner) {
	c.terminalProgressMu.Lock()
	defer c.terminalProgressMu.Unlock()
	if c.terminalProgressOwner != owner {
		return
	}
	c.outputMu.Lock()
	_, _ = writeConsoleString(c.stdout, terminalProgressSequence(terminalProgressStateClear, 0))
	c.outputMu.Unlock()
	c.terminalProgressOwner = nil
}

// terminalProgressSequence encodes the OSC 9;4 protocol understood by supporting terminal emulators.
func terminalProgressSequence(state, progress int) string {
	progress = max(min(progress, 100), 0)
	return fmt.Sprintf("\x1b]9;4;%d;%d\x07", state, progress)
}
