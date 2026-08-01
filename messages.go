package console

import (
	"fmt"
	"io"
	"strings"
)

// ANSI style and color codes are grouped here so callers can compose them with Style.
const (
	// ColorReset resets ANSI styling.
	//
	// Example: inspect the reset sequence
	//
	//	fmt.Printf("%q\n", console.ColorReset)
	//	// "\x1b[0m"
	ColorReset = "\033[0m"
	// StyleBold enables bold ANSI text.
	//
	// Example: inspect the bold sequence
	//
	//	fmt.Printf("%q\n", console.StyleBold)
	//	// "\x1b[1m"
	StyleBold = "\033[1m"
	// StyleDim enables dim ANSI text.
	//
	// Example: inspect the dim sequence
	//
	//	fmt.Printf("%q\n", console.StyleDim)
	//	// "\x1b[2m"
	StyleDim = "\033[2m"
	// StyleUnderline enables underlined ANSI text.
	//
	// Example: inspect the underline sequence
	//
	//	fmt.Printf("%q\n", console.StyleUnderline)
	//	// "\x1b[4m"
	StyleUnderline = "\033[4m"
	// ColorBlack is a black ANSI foreground color.
	//
	// Example: inspect the black sequence
	//
	//	fmt.Printf("%q\n", console.ColorBlack)
	//	// "\x1b[30m"
	ColorBlack = "\033[30m"
	// ColorRed is a red ANSI foreground color.
	//
	// Example: inspect the red sequence
	//
	//	fmt.Printf("%q\n", console.ColorRed)
	//	// "\x1b[31m"
	ColorRed = "\033[31m"
	// ColorGreen is a green ANSI foreground color.
	//
	// Example: inspect the green sequence
	//
	//	fmt.Printf("%q\n", console.ColorGreen)
	//	// "\x1b[32m"
	ColorGreen = "\033[32m"
	// ColorYellow is a yellow ANSI foreground color.
	//
	// Example: inspect the yellow sequence
	//
	//	fmt.Printf("%q\n", console.ColorYellow)
	//	// "\x1b[33m"
	ColorYellow = "\033[33m"
	// ColorBlue is a blue ANSI foreground color.
	//
	// Example: inspect the blue sequence
	//
	//	fmt.Printf("%q\n", console.ColorBlue)
	//	// "\x1b[34m"
	ColorBlue = "\033[34m"
	// ColorMagenta is a magenta ANSI foreground color.
	//
	// Example: inspect the magenta sequence
	//
	//	fmt.Printf("%q\n", console.ColorMagenta)
	//	// "\x1b[35m"
	ColorMagenta = "\033[35m"
	// ColorCyan is a cyan ANSI foreground color.
	//
	// Example: inspect the cyan sequence
	//
	//	fmt.Printf("%q\n", console.ColorCyan)
	//	// "\x1b[36m"
	ColorCyan = "\033[36m"
	// ColorWhite is a white ANSI foreground color.
	//
	// Example: inspect the white sequence
	//
	//	fmt.Printf("%q\n", console.ColorWhite)
	//	// "\x1b[37m"
	ColorWhite = "\033[37m"
	// ColorGray is a muted gray ANSI foreground color.
	//
	// Example: inspect the gray sequence
	//
	//	fmt.Printf("%q\n", console.ColorGray)
	//	// "\x1b[90m"
	ColorGray = "\033[90m"
	// ColorBoldWhite is a bold white ANSI foreground color.
	//
	// Example: inspect the bold white sequence
	//
	//	fmt.Printf("%q\n", console.ColorBoldWhite)
	//	// "\x1b[1;97m"
	ColorBoldWhite = "\033[1;97m"
	// ColorBoldGreen is a bold green ANSI foreground color.
	//
	// Example: inspect the bold green sequence
	//
	//	fmt.Printf("%q\n", console.ColorBoldGreen)
	//	// "\x1b[1;32m"
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
//
// Example: print without a newline
//
//	var output bytes.Buffer
//	console.SetDefault(console.New(console.Config{Stdout: &output}))
//	console.Print("deploying")
//	fmt.Printf("%q\n", output.String())
//	// "deploying"
func Print(values ...any) { Default().Print(values...) }

// Printf writes formatted output through the default console without adding a newline.
//
// Example: print formatted output
//
//	var output bytes.Buffer
//	console.SetDefault(console.New(console.Config{Stdout: &output}))
//	console.Printf("copied %d files", 3)
//	fmt.Printf("%q\n", output.String())
//	// "copied 3 files"
func Printf(format string, arguments ...any) { Default().Printf(format, arguments...) }

// Println writes values through the default console followed by a newline.
//
// Example: print a line
//
//	console.Println("deployment complete")
//	// deployment complete
func Println(values ...any) { Default().Println(values...) }

// NewLine writes one blank line through the default console.
//
// Example: separate output
//
//	console.Println("before")
//	console.NewLine()
//	console.Println("after")
//	// before
//	//
//	// after
func NewLine() { Default().NewLine() }

// StdoutWriter returns a coordinated writer using a snapshot of the current default console.
// Later calls to SetDefault do not retarget an existing writer.
//
// Example: pass console output to an io.Writer API
//
//	fmt.Fprintln(console.StdoutWriter(), "download complete")
//	// download complete
func StdoutWriter() io.Writer { return Default().StdoutWriter() }

// StderrWriter returns a coordinated writer using a snapshot of the current default console.
// Later calls to SetDefault do not retarget an existing writer.
//
// Example: pass console errors to an io.Writer API
//
//	fmt.Fprintln(console.StderrWriter(), "download failed")
//	// download failed
func StderrWriter() io.Writer { return Default().StderrWriter() }

// ActionMark returns the default console's action indicator.
//
// Example: inspect the action mark
//
//	fmt.Println(console.ActionMark())
//	// ·
func ActionMark() string { return Default().ActionMark() }

// InfoMark returns the default console's informational indicator.
//
// Example: inspect the information mark
//
//	fmt.Println(console.InfoMark())
//	// ·
func InfoMark() string { return Default().InfoMark() }

// SuccessMark returns the default console's success indicator.
//
// Example: inspect the success mark
//
//	fmt.Println(console.SuccessMark())
//	// ✔
func SuccessMark() string { return Default().SuccessMark() }

// WarnMark returns the default console's warning indicator.
//
// Example: inspect the warning mark
//
//	fmt.Println(console.WarnMark())
//	// !
func WarnMark() string { return Default().WarnMark() }

// ErrorMark returns the default console's error indicator.
//
// Example: inspect the error mark
//
//	fmt.Println(console.ErrorMark())
//	// ✖
func ErrorMark() string { return Default().ErrorMark() }

// DebugMark returns the default console's debug indicator.
//
// Example: inspect the debug mark
//
//	fmt.Println(console.DebugMark())
//	// ?
func DebugMark() string { return Default().DebugMark() }

// Action prints an action message through the default console.
//
// Example: announce work
//
//	console.Action("building release")
//	// · building release
func Action(message string) { Default().Action(message) }

// Actionf prints a formatted action message through the default console.
//
// Example: announce formatted work
//
//	console.Actionf("building %s", "release")
//	// · building release
func Actionf(format string, arguments ...any) { Default().Actionf(format, arguments...) }

// Info prints an informational message through the default console.
//
// Example: share information
//
//	console.Info("using cached dependencies")
//	// · using cached dependencies
func Info(message string) { Default().Info(message) }

// Infof prints a formatted informational message through the default console.
//
// Example: share formatted information
//
//	console.Infof("using %s dependencies", "cached")
//	// · using cached dependencies
func Infof(format string, arguments ...any) { Default().Infof(format, arguments...) }

// Success prints a success message through the default console.
//
// Example: report success
//
//	console.Success("release published")
//	// ✔ release published
func Success(message string) { Default().Success(message) }

// Successf prints a formatted success message through the default console.
//
// Example: report formatted success
//
//	console.Successf("published %s", "v1.2.0")
//	// ✔ published v1.2.0
func Successf(format string, arguments ...any) { Default().Successf(format, arguments...) }

// Warn prints a warning message through the default console.
//
// Example: report a warning
//
//	console.Warn("configuration is deprecated")
//	// ! configuration is deprecated
func Warn(message string) { Default().Warn(message) }

// Warnf prints a formatted warning message through the default console.
//
// Example: report a formatted warning
//
//	console.Warnf("retrying in %d seconds", 5)
//	// ! retrying in 5 seconds
func Warnf(format string, arguments ...any) { Default().Warnf(format, arguments...) }

// Error prints an error message through the default console.
//
// Example: report an error
//
//	console.Error("deployment failed")
//	// ✖ deployment failed
func Error(message string) { Default().Error(message) }

// Errorf prints a formatted error message through the default console.
//
// Example: report a formatted error
//
//	console.Errorf("deployment failed: %s", "timeout")
//	// ✖ deployment failed: timeout
func Errorf(format string, arguments ...any) { Default().Errorf(format, arguments...) }

// Fatal prints an error through the default console and exits with status 1.
//
// Example: report a fatal error
//
//	console.SetDefault(console.New(console.Config{
//		Exit: func(code int) { fmt.Println("exit", code) },
//	}))
//	console.Fatal("invalid configuration")
//	// ✖ invalid configuration
//	// exit 1
func Fatal(message string) { Default().Fatal(message) }

// Fatalf prints a formatted error through the default console and exits with status 1.
//
// Example: report a formatted fatal error
//
//	console.SetDefault(console.New(console.Config{
//		Exit: func(code int) { fmt.Println("exit", code) },
//	}))
//	console.Fatalf("invalid port: %d", 0)
//	// ✖ invalid port: 0
//	// exit 1
func Fatalf(format string, arguments ...any) { Default().Fatalf(format, arguments...) }

// Debug prints a diagnostic message through the default console when enabled.
//
// Example: print diagnostics
//
//	debug := true
//	console.SetDefault(console.New(console.Config{DebugEnabled: &debug}))
//	console.Debug("cache miss")
//	// ? cache miss
func Debug(message string) { Default().Debug(message) }

// Debugf prints a formatted diagnostic message through the default console when enabled.
//
// Example: print formatted diagnostics
//
//	debug := true
//	console.SetDefault(console.New(console.Config{DebugEnabled: &debug}))
//	console.Debugf("attempt %d of %d", 1, 3)
//	// ? attempt 1 of 3
func Debugf(format string, arguments ...any) { Default().Debugf(format, arguments...) }

// Style applies ANSI styles using the default console's color policy.
//
// Example: compose text styles
//
//	color := true
//	console.SetDefault(console.New(console.Config{ColorEnabled: &color}))
//	fmt.Printf("%q\n", console.Style("ready", console.StyleBold, console.ColorGreen))
//	// "\x1b[1m\x1b[32mready\x1b[0m"
func Style(value string, styles ...string) string { return Default().Style(value, styles...) }

// Colorize applies an ANSI color using the default console's color policy.
//
// Example: color text
//
//	color := true
//	console.SetDefault(console.New(console.Config{ColorEnabled: &color}))
//	fmt.Printf("%q\n", console.Colorize(console.ColorCyan, "connected"))
//	// "\x1b[36mconnected\x1b[0m"
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
