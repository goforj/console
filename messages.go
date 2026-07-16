package console

import (
	"fmt"
	"io"
	"strings"
)

// ANSI style and color codes are grouped here so callers can compose them with Style.
const (
	// ColorReset resets ANSI styling.
	ColorReset = "\033[0m"
	// StyleBold enables bold ANSI text.
	StyleBold = "\033[1m"
	// StyleDim enables dim ANSI text.
	StyleDim = "\033[2m"
	// StyleUnderline enables underlined ANSI text.
	StyleUnderline = "\033[4m"
	// ColorBlack is a black ANSI foreground color.
	ColorBlack = "\033[30m"
	// ColorRed is a red ANSI foreground color.
	ColorRed = "\033[31m"
	// ColorGreen is a green ANSI foreground color.
	ColorGreen = "\033[32m"
	// ColorYellow is a yellow ANSI foreground color.
	ColorYellow = "\033[33m"
	// ColorBlue is a blue ANSI foreground color.
	ColorBlue = "\033[34m"
	// ColorMagenta is a magenta ANSI foreground color.
	ColorMagenta = "\033[35m"
	// ColorCyan is a cyan ANSI foreground color.
	ColorCyan = "\033[36m"
	// ColorWhite is a white ANSI foreground color.
	ColorWhite = "\033[37m"
	// ColorGray is a muted gray ANSI foreground color.
	ColorGray = "\033[90m"
	// ColorBoldWhite is a bold white ANSI foreground color.
	ColorBoldWhite = "\033[1;97m"
	// ColorBoldGreen is a bold green ANSI foreground color.
	ColorBoldGreen = "\033[1;32m"
)

// Print writes values to ordinary output without adding a newline.
func (c *Console) Print(values ...any) {
	c.write(c.stdout, fmt.Sprint(values...), true)
}

// Printf writes formatted ordinary output without adding a newline.
func (c *Console) Printf(format string, arguments ...any) {
	c.write(c.stdout, fmt.Sprintf(format, arguments...), true)
}

// Println writes values to ordinary output followed by a newline.
func (c *Console) Println(values ...any) {
	c.write(c.stdout, fmt.Sprintln(values...), true)
}

// NewLine writes one blank line to ordinary output.
func (c *Console) NewLine() {
	c.write(c.stdout, "\n", true)
}

// StdoutWriter returns a writer coordinated with this console's prompts and transient displays.
// The destination is captured when the adapter is constructed, and its write results are preserved.
func (c *Console) StdoutWriter() io.Writer {
	return consoleOutputWriter{console: c, destination: c.stdout, stdout: true}
}

// StderrWriter returns a writer coordinated with this console's prompts and transient displays.
// The destination is captured when the adapter is constructed, and its write results are preserved.
func (c *Console) StderrWriter() io.Writer {
	return consoleOutputWriter{console: c, destination: c.stderr}
}

// ActionMark returns the action indicator.
func (c *Console) ActionMark() string {
	return c.mark(c.stdout, ColorGray, c.marks.Action)
}

// InfoMark returns the informational indicator.
func (c *Console) InfoMark() string {
	return c.mark(c.stdout, ColorGray, c.marks.Info)
}

// SuccessMark returns the success indicator.
func (c *Console) SuccessMark() string {
	return c.mark(c.stdout, ColorGreen, c.marks.Success)
}

// WarnMark returns the warning indicator.
func (c *Console) WarnMark() string {
	return c.mark(c.stdout, ColorYellow, c.marks.Warn)
}

// ErrorMark returns the error indicator using the stderr color policy.
func (c *Console) ErrorMark() string {
	return c.mark(c.stderr, ColorRed, c.marks.Error)
}

// DebugMark returns the debug indicator.
func (c *Console) DebugMark() string {
	return c.mark(c.stdout, ColorGray, c.marks.Debug)
}

// Action prints an action message.
func (c *Console) Action(message string) {
	c.message(c.stdout, c.ActionMark(), message, true)
}

// Actionf prints a formatted action message.
func (c *Console) Actionf(format string, arguments ...any) {
	c.Action(fmt.Sprintf(format, arguments...))
}

// Info prints an informational message.
func (c *Console) Info(message string) {
	c.message(c.stdout, c.InfoMark(), message, true)
}

// Infof prints a formatted informational message.
func (c *Console) Infof(format string, arguments ...any) {
	c.Info(fmt.Sprintf(format, arguments...))
}

// Success prints a success message.
func (c *Console) Success(message string) {
	c.message(c.stdout, c.SuccessMark(), message, true)
}

// Successf prints a formatted success message.
func (c *Console) Successf(format string, arguments ...any) {
	c.Success(fmt.Sprintf(format, arguments...))
}

// Warn prints a warning message.
func (c *Console) Warn(message string) {
	c.message(c.stdout, c.WarnMark(), message, true)
}

// Warnf prints a formatted warning message.
func (c *Console) Warnf(format string, arguments ...any) {
	c.Warn(fmt.Sprintf(format, arguments...))
}

// Error prints an error message to stderr.
func (c *Console) Error(message string) {
	c.message(c.stderr, c.ErrorMark(), message, false)
}

// Errorf prints a formatted error message to stderr.
func (c *Console) Errorf(format string, arguments ...any) {
	c.Error(fmt.Sprintf(format, arguments...))
}

// Fatal prints an error message and exits with status 1.
func (c *Console) Fatal(message string) {
	c.Error(message)
	c.exit(1)
}

// Fatalf prints a formatted error message and exits with status 1.
func (c *Console) Fatalf(format string, arguments ...any) {
	c.Fatal(fmt.Sprintf(format, arguments...))
}

// Debug prints a diagnostic message when debug output is enabled.
func (c *Console) Debug(message string) {
	if !c.isDebugEnabled() {
		return
	}
	c.message(c.stdout, c.DebugMark(), message, true)
}

// Debugf prints a formatted diagnostic message when debug output is enabled.
func (c *Console) Debugf(format string, arguments ...any) {
	if !c.isDebugEnabled() {
		return
	}
	c.message(c.stdout, c.DebugMark(), fmt.Sprintf(format, arguments...), true)
}

// Style applies ANSI style sequences to value when color output is enabled.
func (c *Console) Style(value string, styles ...string) string {
	if value == "" || len(styles) == 0 || !c.shouldColor(c.stdout) {
		return value
	}
	return strings.Join(styles, "") + value + ColorReset
}

// Colorize applies one ANSI color to value when color output is enabled.
func (c *Console) Colorize(color, value string) string {
	return c.Style(value, color)
}

// Print writes values through the default console without adding a newline.
func Print(values ...any) { Default().Print(values...) }

// Printf writes formatted output through the default console without adding a newline.
func Printf(format string, arguments ...any) { Default().Printf(format, arguments...) }

// Println writes values through the default console followed by a newline.
func Println(values ...any) { Default().Println(values...) }

// NewLine writes one blank line through the default console.
func NewLine() { Default().NewLine() }

// StdoutWriter returns a coordinated writer using a snapshot of the current default console.
// Later calls to SetDefault do not retarget an existing writer.
func StdoutWriter() io.Writer { return Default().StdoutWriter() }

// StderrWriter returns a coordinated writer using a snapshot of the current default console.
// Later calls to SetDefault do not retarget an existing writer.
func StderrWriter() io.Writer { return Default().StderrWriter() }

// ActionMark returns the default console's action indicator.
func ActionMark() string { return Default().ActionMark() }

// InfoMark returns the default console's informational indicator.
func InfoMark() string { return Default().InfoMark() }

// SuccessMark returns the default console's success indicator.
func SuccessMark() string { return Default().SuccessMark() }

// WarnMark returns the default console's warning indicator.
func WarnMark() string { return Default().WarnMark() }

// ErrorMark returns the default console's error indicator.
func ErrorMark() string { return Default().ErrorMark() }

// DebugMark returns the default console's debug indicator.
func DebugMark() string { return Default().DebugMark() }

// Action prints an action message through the default console.
func Action(message string) { Default().Action(message) }

// Actionf prints a formatted action message through the default console.
func Actionf(format string, arguments ...any) { Default().Actionf(format, arguments...) }

// Info prints an informational message through the default console.
func Info(message string) { Default().Info(message) }

// Infof prints a formatted informational message through the default console.
func Infof(format string, arguments ...any) { Default().Infof(format, arguments...) }

// Success prints a success message through the default console.
func Success(message string) { Default().Success(message) }

// Successf prints a formatted success message through the default console.
func Successf(format string, arguments ...any) { Default().Successf(format, arguments...) }

// Warn prints a warning message through the default console.
func Warn(message string) { Default().Warn(message) }

// Warnf prints a formatted warning message through the default console.
func Warnf(format string, arguments ...any) { Default().Warnf(format, arguments...) }

// Error prints an error message through the default console.
func Error(message string) { Default().Error(message) }

// Errorf prints a formatted error message through the default console.
func Errorf(format string, arguments ...any) { Default().Errorf(format, arguments...) }

// Fatal prints an error through the default console and exits with status 1.
func Fatal(message string) { Default().Fatal(message) }

// Fatalf prints a formatted error through the default console and exits with status 1.
func Fatalf(format string, arguments ...any) { Default().Fatalf(format, arguments...) }

// Debug prints a diagnostic message through the default console when enabled.
func Debug(message string) { Default().Debug(message) }

// Debugf prints a formatted diagnostic message through the default console when enabled.
func Debugf(format string, arguments ...any) { Default().Debugf(format, arguments...) }

// Style applies ANSI styles using the default console's color policy.
func Style(value string, styles ...string) string { return Default().Style(value, styles...) }

// Colorize applies an ANSI color using the default console's color policy.
func Colorize(color, value string) string { return Default().Colorize(color, value) }

// consoleOutputWriter adapts coordinated console output to APIs that accept io.Writer.
type consoleOutputWriter struct {
	console     *Console
	destination io.Writer
	stdout      bool
}

// Write coordinates one caller write and preserves the configured destination's result.
func (w consoleOutputWriter) Write(value []byte) (int, error) {
	if len(value) == 0 {
		return 0, nil
	}
	w.console.sessionMu.RLock()
	written, err := w.console.writeCoordinated(w.destination, string(value), w.stdout)
	w.console.sessionMu.RUnlock()
	return written, err
}

// message writes one sanitized and balanced semantic message under the console output lock.
func (c *Console) message(writer io.Writer, mark, message string, stdout bool) {
	c.write(writer, c.renderSemanticMessage(mark, message), stdout)
}

// renderSemanticMessage removes terminal controls while preserving safe styling and hanging indentation.
func (c *Console) renderSemanticMessage(mark, message string) string {
	lines := strings.Split(sanitizeLayoutText(message, true), "\n")
	lines = balanceANSILines(lines)
	indent := strings.Repeat(" ", VisibleWidth(mark)+1)
	return mark + " " + strings.Join(lines, "\n"+indent) + "\n"
}

// mark styles one semantic symbol according to its destination's color capability.
func (c *Console) mark(writer io.Writer, color, symbol string) string {
	symbol = singleLineLayoutText(symbol)
	if symbol == "" || !c.shouldColor(writer) {
		return symbol
	}
	return color + symbol + ColorReset
}

// isDebugEnabled resolves the explicit override before the framework and conventional debug variables.
func (c *Console) isDebugEnabled() bool {
	if c.debugEnabled != nil {
		return *c.debugEnabled
	}
	for _, key := range []string{"FORJ_DEBUG", "APP_DEBUG", "DEBUG"} {
		if environmentFlag(c.getenv(key)) {
			return true
		}
	}
	return false
}
