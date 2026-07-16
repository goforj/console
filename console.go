package console

import (
	"bufio"
	"errors"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	defaultTerminalWidth  = 80
	maximumTerminalWidth  = 32768
	defaultLoaderInterval = 80 * time.Millisecond
)

// Config configures a Console instance.
//
// Every field is optional. Nil functions and writers use their operating-system
// defaults, while nil boolean pointers select automatic behavior.
//
// Example: configure a fixed output width
//
//	configuration := console.Config{Width: 100}
//	commandConsole := console.New(configuration)
//	fmt.Println(commandConsole.Width())
//	// 100
type Config struct {
	// Stdin supplies answers to prompts.
	Stdin io.Reader
	// Stdout receives ordinary and successful output.
	Stdout io.Writer
	// Stderr receives errors and fatal messages.
	Stderr io.Writer

	// ColorEnabled forces ANSI styling on or off. Nil enables environment and terminal detection.
	ColorEnabled *bool
	// DebugEnabled forces debug messages on or off. Nil reads the supported debug environment variables.
	DebugEnabled *bool
	// InteractiveEnabled overrides terminal detection for prompt-oriented callers.
	InteractiveEnabled *bool
	// UnicodeEnabled selects Unicode or ASCII presentation characters. Nil enables conservative environment detection.
	UnicodeEnabled *bool
	// AnimationsEnabled permits or disables transient loader and progress output. Even when true, stdout must be a terminal.
	AnimationsEnabled *bool

	// Width fixes the available output width. Values less than one use terminal detection and then an 80-column fallback.
	// Configured, detected, and environment widths are capped at 32,768 columns to keep layout allocations practical.
	Width int
	// LoaderInterval controls animated loader frame timing. Values less than or equal to zero use 80 milliseconds.
	LoaderInterval time.Duration
	// Marks replaces the complete semantic symbol set when non-nil.
	Marks *Marks

	// Getenv reads environment variables.
	Getenv func(string) string
	// IsTerminal reports whether a file descriptor is attached to a terminal.
	IsTerminal func(int) bool
	// GetSize returns the terminal dimensions for a file descriptor.
	GetSize func(int) (width, height int, err error)
	// Exit terminates the process for Fatal and Fatalf.
	Exit func(int)
	// ReadSecret reads one value without echoing it to the terminal.
	// Nil uses terminal password input; tests and custom terminals can inject a reader.
	ReadSecret func() (string, error)
}

// Marks contains the symbols used for messages, lists, selections, and loaders.
//
// Example: define custom semantic marks
//
//	marks := console.Marks{Success: "OK"}
//	fmt.Println(marks.Success)
//	// OK
type Marks struct {
	// Action identifies work that is starting or underway.
	Action string
	// Info identifies neutral information.
	Info string
	// Success identifies successful work.
	Success string
	// Warn identifies a warning.
	Warn string
	// Error identifies a failure.
	Error string
	// Debug identifies diagnostic output.
	Debug string
	// Bullet identifies an unordered list item.
	Bullet string
	// Pointer identifies a prompt or selected item.
	Pointer string
	// SpinnerFrames contains the loader animation sequence.
	SpinnerFrames []string
}

// DefaultMarks returns the Unicode symbols used by a default console.
//
// Example:
//
//	marks := console.DefaultMarks()
//	fmt.Println(marks.Success, marks.Warn, marks.Error)
//	// ✔ ! ✖
func DefaultMarks() Marks {
	return Marks{
		Action:        "·",
		Info:          "·",
		Success:       "✔",
		Warn:          "!",
		Error:         "✖",
		Debug:         "?",
		Bullet:        "•",
		Pointer:       "›",
		SpinnerFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// ASCIIMarks returns symbols suitable for constrained terminals and plain logs.
//
// Example:
//
//	marks := console.ASCIIMarks()
//	fmt.Println(marks.Success, marks.Warn, marks.Error)
//	// + ! x
func ASCIIMarks() Marks {
	return Marks{
		Action:        "-",
		Info:          "i",
		Success:       "+",
		Warn:          "!",
		Error:         "x",
		Debug:         "?",
		Bullet:        "-",
		Pointer:       ">",
		SpinnerFrames: []string{"|", "/", "-", "\\"},
	}
}

// Console coordinates output policy, terminal capabilities, prompts, and transient displays.
// A Console is safe for concurrent message writes and must be constructed with New.
//
// Example: declare an isolated console
//
//	var commandConsole *console.Console = console.New(console.Config{Width: 120})
//	fmt.Println(commandConsole.Width())
//	// 120
type Console struct {
	stdin              *bufio.Reader
	stdinSource        io.Reader
	stdout             io.Writer
	stderr             io.Writer
	stderrSharesStdout bool

	colorEnabled       *bool
	debugEnabled       *bool
	interactiveEnabled *bool
	unicodeEnabled     bool
	animationsEnabled  *bool

	width          int
	loaderInterval time.Duration
	marks          Marks

	getenv       func(string) string
	isTerminal   func(int) bool
	supportsANSI func(int) bool
	getSize      func(int) (width, height int, err error)
	exit         func(int)
	readSecret   func() (string, error)
	newTicker    func(time.Duration) loaderTicker

	inputMu      sync.Mutex
	sessionMu    sync.RWMutex
	outputMu     sync.Mutex
	transientMu  sync.Mutex
	active       transientOwner
	partialLine  bool
	promptActive bool
}

var defaultState = struct {
	sync.RWMutex
	console *Console
}{console: New(Config{})}

// New creates an isolated console with optional runtime overrides.
//
// Example:
//
//	var output bytes.Buffer
//	color := false
//	unicode := true
//	commandConsole := console.New(console.Config{
//		Stdout:         &output,
//		ColorEnabled:   &color,
//		UnicodeEnabled: &unicode,
//	})
//	commandConsole.Success("ready")
//	fmt.Print(output.String())
//	// ✔ ready
func New(config Config) *Console {
	stdin := config.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := config.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := config.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	getenv := config.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	isTerminal := config.IsTerminal
	supportsANSI := terminalSupportsANSI
	if isTerminal == nil {
		isTerminal = term.IsTerminal
	} else {
		// A caller-provided terminal detector owns the descriptor contract, which may
		// represent a virtual terminal that native operating-system probes cannot inspect.
		supportsANSI = func(int) bool { return true }
	}
	getSize := config.GetSize
	if getSize == nil {
		getSize = term.GetSize
	}
	exit := config.Exit
	if exit == nil {
		exit = os.Exit
	}
	readSecret := config.ReadSecret
	if readSecret == nil {
		readSecret = func() (string, error) {
			return readTerminalSecret(stdin)
		}
	}

	unicodeEnabled := detectUnicode(config.UnicodeEnabled, getenv)
	marks := DefaultMarks()
	if !unicodeEnabled {
		marks = ASCIIMarks()
	}
	if config.Marks != nil {
		marks = cloneMarks(*config.Marks)
	}

	loaderInterval := config.LoaderInterval
	if loaderInterval <= 0 {
		loaderInterval = defaultLoaderInterval
	}

	return &Console{
		stdin:              bufio.NewReader(stdin),
		stdinSource:        stdin,
		stdout:             stdout,
		stderr:             stderr,
		stderrSharesStdout: sameWriter(stdout, stderr),
		colorEnabled:       cloneBool(config.ColorEnabled),
		debugEnabled:       cloneBool(config.DebugEnabled),
		interactiveEnabled: cloneBool(config.InteractiveEnabled),
		unicodeEnabled:     unicodeEnabled,
		animationsEnabled:  cloneBool(config.AnimationsEnabled),
		width:              config.Width,
		loaderInterval:     loaderInterval,
		marks:              marks,
		getenv:             getenv,
		isTerminal:         isTerminal,
		supportsANSI:       supportsANSI,
		getSize:            getSize,
		exit:               exit,
		readSecret:         readSecret,
		newTicker:          newRealLoaderTicker,
	}
}

// SetDefault replaces the console used by package-level helpers.
// It panics when console is nil because package helpers always require a usable runtime.
//
// Example:
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//
//	var output bytes.Buffer
//	console.SetDefault(console.New(console.Config{Stdout: &output}))
//	console.Println("ready")
//	fmt.Print(output.String())
//	// ready
func SetDefault(console *Console) {
	if console == nil {
		panic("console: default console cannot be nil")
	}

	defaultState.Lock()
	defaultState.console = console
	defaultState.Unlock()
}

// Default returns the console currently used by package-level helpers.
//
// Example:
//
//	fmt.Println(console.Default() != nil)
//	// true
func Default() *Console {
	defaultState.RLock()
	console := defaultState.console
	defaultState.RUnlock()
	return console
}

// Width returns the configured or detected terminal width, capped at 32,768 columns, and falls back to 80 columns.
func (c *Console) Width() int {
	if width := practicalTerminalWidth(c.width); width > 0 {
		return width
	}
	if descriptor, ok := writerDescriptor(c.stdout); ok {
		width, _, err := c.getSize(descriptor)
		if err == nil && width > 0 {
			return practicalTerminalWidth(width)
		}
	}
	if width := environmentTerminalWidth(c.getenv("COLUMNS")); width > 0 {
		return width
	}
	return defaultTerminalWidth
}

// IsInteractive reports whether both configured input and output are terminals unless explicitly overridden.
func (c *Console) IsInteractive() bool {
	if c.interactiveEnabled != nil {
		return *c.interactiveEnabled
	}
	if isCIEnvironment(c.getenv) {
		return false
	}
	inputDescriptor, inputOK := readerDescriptor(c.stdinSource)
	outputDescriptor, outputOK := writerDescriptor(c.stdout)
	return inputOK && outputOK && c.isTerminal(inputDescriptor) && c.isTerminal(outputDescriptor)
}

// SupportsColor reports whether ordinary output should contain ANSI styling.
func (c *Console) SupportsColor() bool {
	return c.shouldColor(c.stdout)
}

// SupportsUnicode reports whether the console selected Unicode presentation characters.
func (c *Console) SupportsUnicode() bool {
	return c.unicodeEnabled
}

// Width returns the width of the default console.
//
// Example:
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	console.SetDefault(console.New(console.Config{Width: 100}))
//
//	fmt.Println(console.Width())
//	// 100
func Width() int {
	return Default().Width()
}

// IsInteractive reports whether the default console is interactive.
//
// Example:
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	interactive := true
//	console.SetDefault(console.New(console.Config{InteractiveEnabled: &interactive}))
//
//	fmt.Println(console.IsInteractive())
//	// true
func IsInteractive() bool {
	return Default().IsInteractive()
}

// SupportsColor reports whether the default console emits ANSI styling.
//
// Example:
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	color := true
//	console.SetDefault(console.New(console.Config{ColorEnabled: &color}))
//
//	fmt.Println(console.SupportsColor())
//	// true
func SupportsColor() bool {
	return Default().SupportsColor()
}

// SupportsUnicode reports whether the default console uses Unicode presentation characters.
//
// Example:
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	unicode := false
//	console.SetDefault(console.New(console.Config{UnicodeEnabled: &unicode}))
//
//	fmt.Println(console.SupportsUnicode())
//	// false
func SupportsUnicode() bool {
	return Default().SupportsUnicode()
}

// cloneBool copies an optional override so later caller mutation cannot race with output.
func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// cloneMarks keeps mutable spinner slices private to a console instance.
func cloneMarks(marks Marks) Marks {
	marks.SpinnerFrames = append([]string(nil), marks.SpinnerFrames...)
	return marks
}

// detectUnicode chooses ASCII only for an explicit override or a clearly constrained environment.
func detectUnicode(override *bool, getenv func(string) string) bool {
	if override != nil {
		return *override
	}
	if strings.EqualFold(strings.TrimSpace(getenv("TERM")), "dumb") {
		return false
	}
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		locale := strings.TrimSpace(getenv(key))
		if locale == "" {
			continue
		}
		upper := strings.ToUpper(locale)
		if upper == "C" || upper == "POSIX" {
			return false
		}
		return strings.Contains(upper, "UTF-8") || strings.Contains(upper, "UTF8")
	}
	return true
}

// shouldColor applies explicit, environment, and terminal policies in that order.
func (c *Console) shouldColor(writer io.Writer) bool {
	if c.colorEnabled != nil {
		return *c.colorEnabled
	}
	if c.getenv("NO_COLOR") != "" {
		return false
	}
	if environmentFlag(c.getenv("CLICOLOR_FORCE")) {
		return true
	}
	if c.getenv("CLICOLOR") == "0" || strings.EqualFold(c.getenv("TERM"), "dumb") {
		return false
	}
	descriptor, ok := writerDescriptor(writer)
	return ok && c.isTerminal(descriptor) && c.supportsANSI(descriptor)
}

// shouldAnimate prevents transient carriage-return output from leaking into redirected logs.
func (c *Console) shouldAnimate() bool {
	if c.animationsEnabled != nil && !*c.animationsEnabled {
		return false
	}
	if c.animationsEnabled == nil {
		if strings.EqualFold(strings.TrimSpace(c.getenv("TERM")), "dumb") || isCIEnvironment(c.getenv) {
			return false
		}
	}
	descriptor, ok := writerDescriptor(c.stdout)
	if !ok || !c.isTerminal(descriptor) || !c.supportsANSI(descriptor) {
		return false
	}
	return c.animationsEnabled == nil || *c.animationsEnabled
}

// practicalTerminalWidth bounds allocations driven by a configured or detected terminal width.
// The cap is far beyond practical terminal sizes while preventing hostile hooks from requesting
// near-address-space-sized rules, tables, or boxes.
func practicalTerminalWidth(width int) int {
	if width < 1 {
		return 0
	}
	return min(width, maximumTerminalWidth)
}

// environmentTerminalWidth parses COLUMNS without allowing integer overflow or impractical allocations.
func environmentTerminalWidth(value string) int {
	width, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || width == 0 {
		return 0
	}
	if width > maximumTerminalWidth {
		return maximumTerminalWidth
	}
	return int(width)
}

// isCIEnvironment recognizes conventional truthy CI values without treating explicit false values as enabled.
func isCIEnvironment(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv("CI"))) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// environmentFlag follows common CLI semantics where only an empty value or zero is disabled.
func environmentFlag(value string) bool {
	return value != "" && value != "0"
}

// descriptor exposes the common file-descriptor capability without requiring an os.File.
type descriptor interface {
	Fd() uintptr
}

// writerDescriptor returns a writer descriptor when terminal detection is possible.
func writerDescriptor(writer io.Writer) (int, bool) {
	value, ok := writer.(descriptor)
	if !ok {
		return 0, false
	}
	return int(value.Fd()), true
}

// readerDescriptor returns a reader descriptor when terminal detection is possible.
func readerDescriptor(reader io.Reader) (int, bool) {
	value, ok := reader.(descriptor)
	if !ok {
		return 0, false
	}
	return int(value.Fd()), true
}

// sameWriter identifies the common configuration where stdout and stderr share one comparable writer.
func sameWriter(first, second io.Writer) bool {
	firstType := reflect.TypeOf(first)
	if firstType == nil || firstType != reflect.TypeOf(second) || !firstType.Comparable() {
		return false
	}
	return first == second
}

// write prevents ordinary output from overwriting a prompt while it waits for input.
func (c *Console) write(writer io.Writer, value string, stdout bool) {
	if value == "" {
		return
	}
	c.sessionMu.RLock()
	_, _ = c.writeCoordinated(writer, value, stdout)
	c.sessionMu.RUnlock()
}

// writeCoordinated serializes output and preserves an active transient line around durable writes.
// The caller must hold either side of sessionMu so prompt ownership cannot change mid-write.
func (c *Console) writeCoordinated(writer io.Writer, value string, stdout bool) (int, error) {
	c.transientMu.Lock()
	defer c.transientMu.Unlock()
	c.outputMu.Lock()
	defer c.outputMu.Unlock()
	if c.active != nil && !c.partialLine {
		if _, err := writeConsoleString(c.stdout, clearTransientLine); err != nil {
			return 0, err
		}
	}
	written, writeErr := writeConsoleString(writer, value)
	visibleWritten := max(min(written, len(value)), 0)
	if (stdout || c.stderrSharesStdout) && visibleWritten > 0 {
		c.partialLine = !strings.HasSuffix(value[:visibleWritten], "\n")
	}
	var redrawErr error
	if c.active != nil && !c.partialLine {
		_, redrawErr = writeConsoleString(c.stdout, c.active.renderTransient())
	}
	return written, errors.Join(writeErr, redrawErr)
}

// resumeTransient completes unterminated input and redraws a live display after an interactive prompt returns.
func (c *Console) resumeTransient(lineTerminated bool) error {
	c.transientMu.Lock()
	defer c.transientMu.Unlock()
	c.promptActive = false
	if !c.partialLine {
		return nil
	}
	c.outputMu.Lock()
	defer c.outputMu.Unlock()
	if !lineTerminated {
		written, err := writeConsoleString(c.stdout, "\n")
		if written < 1 {
			return err
		}
		c.partialLine = false
		if err != nil {
			if c.active == nil {
				return err
			}
			_, redrawErr := writeConsoleString(c.stdout, c.active.renderTransient())
			return errors.Join(err, redrawErr)
		}
	}
	c.partialLine = false
	if c.active == nil {
		return nil
	}
	_, err := writeConsoleString(c.stdout, c.active.renderTransient())
	return err
}

// writeConsoleString restores the io.Writer contract when a destination silently accepts only a prefix.
func writeConsoleString(writer io.Writer, value string) (int, error) {
	written, err := writeTerminalString(writer, value)
	if written < 0 {
		written = 0
		err = errors.Join(err, io.ErrShortWrite)
	} else if written > len(value) {
		written = len(value)
		err = errors.Join(err, io.ErrShortWrite)
	}
	if written != len(value) && err == nil {
		err = io.ErrShortWrite
	}
	return written, err
}
