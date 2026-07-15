package console

import (
	"errors"
	"io"
	"strings"
)

const clearTransientLine = "\r\x1b[2K"

// ErrTransientActive is returned when another live loader or progress display owns the transient line.
// @group Terminal
var ErrTransientActive = errors.New("console: another transient display is already active")

// transientOwner renders one replaceable line while the console coordinates durable output.
type transientOwner interface {
	renderTransient() string
}

// normalizeTransientMessage reduces live display labels to one balanced, safe physical line.
func normalizeTransientMessage(message string) string {
	message = sanitizeLayoutText(message, false)
	message = strings.Join(strings.Fields(message), " ")
	return balanceANSILines([]string{message})[0]
}

// acquireTransient grants one owner exclusive access to the replaceable output line.
func (c *Console) acquireTransient(owner transientOwner) error {
	c.transientMu.Lock()
	defer c.transientMu.Unlock()
	if c.active != nil && c.active != owner {
		return ErrTransientActive
	}
	c.active = owner
	return nil
}

// renderTransient redraws owner only while it still controls an otherwise complete output line.
func (c *Console) renderTransient(owner transientOwner) {
	c.transientMu.Lock()
	defer c.transientMu.Unlock()
	if c.active != owner || c.partialLine {
		return
	}
	c.outputMu.Lock()
	_, _ = io.WriteString(c.stdout, owner.renderTransient())
	c.outputMu.Unlock()
}

// releaseTransient clears the replaceable line and relinquishes ownership after live work has stopped.
func (c *Console) releaseTransient(owner transientOwner, durableOutcome bool) {
	c.transientMu.Lock()
	defer c.transientMu.Unlock()
	if c.active != owner {
		return
	}
	c.outputMu.Lock()
	if c.partialLine && durableOutcome && !c.promptActive {
		_, _ = io.WriteString(c.stdout, "\n")
		c.partialLine = false
	} else if !c.partialLine {
		_, _ = io.WriteString(c.stdout, clearTransientLine)
	}
	c.active = nil
	c.outputMu.Unlock()
}
