package console

import (
	"errors"
	"fmt"
	"math/bits"
	"strings"
	"sync"
)

// errInvalidProgressTotal reports a total that cannot represent determinate progress.
var errInvalidProgressTotal = errors.New("console: progress total must be greater than zero")

// Progress presents determinate work as one transient terminal line and stable semantic lines in redirected output.
// A Progress is concurrency-safe, single-use, and must be constructed with Console.Progress or NewProgress;
// the first call to Complete, Fail, or Stop wins.
type Progress struct {
	console *Console

	mu      sync.Mutex
	message string
	total   int
	current int
	state   progressState
	dynamic bool
}

// progressState identifies the one-way progress lifecycle.
type progressState uint8

const (
	progressReady progressState = iota
	progressRunning
	progressFinished
)

// Progress constructs a determinate progress display without starting it.
// Start returns an error when total is less than one.
func (c *Console) Progress(total int, message string) *Progress {
	return &Progress{
		console: c,
		message: normalizeTransientMessage(message),
		total:   total,
	}
}

// NewProgress constructs a progress display using a snapshot of the current default console.
// It does not start the display.
func NewProgress(total int, message string) *Progress {
	return Default().Progress(total, message)
}

// Start begins the progress display and is harmless when called more than once.
// Live terminal displays can return ErrTransientActive when another display owns the console.
func (p *Progress) Start() error {
	p.mu.Lock()
	if p.state != progressReady {
		p.mu.Unlock()
		return nil
	}
	if p.total < 1 {
		p.mu.Unlock()
		return errInvalidProgressTotal
	}

	p.dynamic = p.console.shouldAnimate()
	if p.dynamic {
		if err := p.console.acquireTransient(p); err != nil {
			p.mu.Unlock()
			return err
		}
		p.state = progressRunning
		p.mu.Unlock()
		p.console.renderTransient(p)
		return nil
	}

	p.state = progressRunning
	message := p.message
	p.console.Action(message)
	p.mu.Unlock()
	return nil
}

// Set replaces the completed amount and clamps it between zero and the total.
// Reaching the total does not complete the display; Complete records the durable outcome.
func (p *Progress) Set(current int) {
	p.mu.Lock()
	if p.state == progressFinished {
		p.mu.Unlock()
		return
	}
	p.current = clampProgressValue(current, p.total)
	dynamic := p.state == progressRunning && p.dynamic
	p.mu.Unlock()
	if dynamic {
		p.console.renderTransient(p)
	}
}

// Add changes the completed amount by delta and clamps it between zero and the total.
func (p *Progress) Add(delta int) {
	p.mu.Lock()
	if p.state == progressFinished {
		p.mu.Unlock()
		return
	}
	if delta >= 0 {
		if delta >= p.total-p.current {
			p.current = clampProgressValue(p.total, p.total)
		} else {
			p.current += delta
		}
	} else if delta < -p.current {
		p.current = 0
	} else {
		p.current += delta
	}
	dynamic := p.state == progressRunning && p.dynamic
	p.mu.Unlock()
	if dynamic {
		p.console.renderTransient(p)
	}
}

// Step replaces the completed amount and message in one atomic progress update.
// The amount is clamped between zero and the total, and updates after a terminal
// operation are ignored.
func (p *Progress) Step(current int, message string) {
	p.mu.Lock()
	if p.state == progressFinished {
		p.mu.Unlock()
		return
	}
	p.current = clampProgressValue(current, p.total)
	p.message = normalizeTransientMessage(message)
	dynamic := p.state == progressRunning && p.dynamic
	p.mu.Unlock()
	if dynamic {
		p.console.renderTransient(p)
	}
}

// Update changes the progress message and immediately redraws a live terminal display.
// Updates after a terminal operation are ignored.
func (p *Progress) Update(message string) {
	p.mu.Lock()
	if p.state == progressFinished {
		p.mu.Unlock()
		return
	}
	p.message = normalizeTransientMessage(message)
	dynamic := p.state == progressRunning && p.dynamic
	p.mu.Unlock()
	if dynamic {
		p.console.renderTransient(p)
	}
}

// Complete fills and finishes the display with a success message.
// An empty message reuses the current progress message.
func (p *Progress) Complete(message string) {
	p.finish(progressFinishComplete, message)
}

// Fail finishes the display with an error message on stderr.
// An empty message reuses the current progress message.
func (p *Progress) Fail(message string) {
	p.finish(progressFinishFail, message)
}

// Stop removes the transient display without printing a completion message.
func (p *Progress) Stop() {
	p.finish(progressFinishStop, "")
}

// progressFinish identifies one terminal progress outcome.
type progressFinish uint8

const (
	progressFinishStop progressFinish = iota
	progressFinishComplete
	progressFinishFail
)

// finish performs the winning terminal transition before writing its durable outcome.
func (p *Progress) finish(outcome progressFinish, message string) {
	p.mu.Lock()
	if p.state == progressFinished {
		p.mu.Unlock()
		return
	}
	wasRunning := p.state == progressRunning
	dynamic := wasRunning && p.dynamic
	if outcome == progressFinishComplete {
		p.current = clampProgressValue(p.total, p.total)
	}
	if message == "" {
		message = p.message
	} else {
		message = normalizeTransientMessage(message)
	}
	p.state = progressFinished
	p.mu.Unlock()

	if dynamic {
		p.console.releaseTransient(p, outcome != progressFinishStop)
	}

	switch outcome {
	case progressFinishComplete:
		p.console.Success(message)
	case progressFinishFail:
		p.console.Error(message)
	}
}

// renderTransient snapshots one carriage-return frame while the console owns output coordination.
func (p *Progress) renderTransient() string {
	p.mu.Lock()
	if p.state != progressRunning || !p.dynamic {
		p.mu.Unlock()
		return ""
	}
	message := p.message
	current := p.current
	total := p.total
	p.mu.Unlock()

	return clearTransientLine + p.renderLine(message, current, total)
}

// renderLine builds one width-bounded progress line with a compact fallback for narrow terminals.
func (p *Progress) renderLine(message string, current, total int) string {
	width := max(p.console.Width(), 1)
	percent := progressPercent(current, total)
	percentText := fmt.Sprintf("%3d%%", percent)
	if width < 24 {
		percentText = strings.TrimSpace(percentText)
		if message == "" {
			return p.console.truncate(percentText, width)
		}
		messageWidth := width - VisibleWidth(percentText) - 1
		if messageWidth < 1 {
			return p.console.truncate(percentText, width)
		}
		return p.console.truncate(message, messageWidth) + " " + percentText
	}

	overhead := VisibleWidth(percentText) + 4
	available := max(width-overhead, 8)
	messageWidth := min(VisibleWidth(message), max(available-8, 0))
	message = p.console.truncate(message, messageWidth)
	barWidth := min(24, available-messageWidth)
	if barWidth < 8 {
		barWidth = 8
	}

	filledWidth := percent * barWidth / 100
	filled, empty := "█", "░"
	if !p.console.unicodeEnabled {
		filled, empty = "=", "-"
	}
	bar := p.console.Colorize(ColorGreen, strings.Repeat(filled, filledWidth)) +
		p.console.Colorize(ColorGray, strings.Repeat(empty, barWidth-filledWidth))
	value := "[" + bar + "] " + percentText
	if message != "" {
		value = message + " " + value
	}
	return value
}

// progressPercent converts a clamped amount to a stable whole-number percentage.
func progressPercent(current, total int) int {
	if total < 1 || current <= 0 {
		return 0
	}
	if current >= total {
		return 100
	}
	high, low := bits.Mul64(uint64(current), 100)
	percent, _ := bits.Div64(high, low, uint64(total))
	return min(int(percent), 99)
}

// clampProgressValue keeps an amount inside a determinate progress range.
func clampProgressValue(current, total int) int {
	if current < 0 || total < 1 {
		return 0
	}
	if current > total {
		return total
	}
	return current
}
