package console

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

// descriptorBuffer provides an in-memory writer with a terminal-like file descriptor.
type descriptorBuffer struct {
	bytes.Buffer
	descriptor uintptr
}

// Fd returns the descriptor used by terminal capability tests.
func (b *descriptorBuffer) Fd() uintptr {
	return b.descriptor
}

// descriptorReader provides an in-memory reader with a terminal-like file descriptor.
type descriptorReader struct {
	io.Reader
	descriptor uintptr
}

// Fd returns the descriptor used by terminal capability tests.
func (r *descriptorReader) Fd() uintptr {
	return r.descriptor
}

// boolPointer returns an independently addressable boolean for configuration overrides.
func boolPointer(value bool) *bool {
	return &value
}

// getenvFrom returns an environment reader isolated to the supplied values.
func getenvFrom(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

// TestMarkFactoriesReturnDocumentedSymbols guards the complete built-in symbol sets.
func TestMarkFactoriesReturnDocumentedSymbols(t *testing.T) {
	t.Parallel()

	wantUnicode := Marks{
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
	wantASCII := Marks{
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

	if got := DefaultMarks(); !reflect.DeepEqual(got, wantUnicode) {
		t.Fatalf("DefaultMarks() = %#v, want %#v", got, wantUnicode)
	}
	if got := ASCIIMarks(); !reflect.DeepEqual(got, wantASCII) {
		t.Fatalf("ASCIIMarks() = %#v, want %#v", got, wantASCII)
	}

	unicodeMarks := DefaultMarks()
	unicodeMarks.SpinnerFrames[0] = "changed"
	if got := DefaultMarks().SpinnerFrames[0]; got != "⠋" {
		t.Fatalf("DefaultMarks().SpinnerFrames[0] = %q after caller mutation, want %q", got, "⠋")
	}

	asciiMarks := ASCIIMarks()
	asciiMarks.SpinnerFrames[0] = "changed"
	if got := ASCIIMarks().SpinnerFrames[0]; got != "|" {
		t.Fatalf("ASCIIMarks().SpinnerFrames[0] = %q after caller mutation, want %q", got, "|")
	}
}

// TestNewCopiesMutableConfiguration verifies that callers cannot mutate a configured console after construction.
func TestNewCopiesMutableConfiguration(t *testing.T) {
	t.Parallel()

	colorEnabled := true
	debugEnabled := true
	interactiveEnabled := true
	unicodeEnabled := true
	animationsEnabled := true
	marks := Marks{
		Action:        "action",
		Info:          "info",
		Success:       "success",
		Warn:          "warn",
		Error:         "error",
		Debug:         "debug",
		Bullet:        "bullet",
		Pointer:       "pointer",
		SpinnerFrames: []string{"one", "two"},
	}
	stdout := &descriptorBuffer{descriptor: 11}
	console := New(Config{
		Stdout:             stdout,
		ColorEnabled:       &colorEnabled,
		DebugEnabled:       &debugEnabled,
		InteractiveEnabled: &interactiveEnabled,
		UnicodeEnabled:     &unicodeEnabled,
		AnimationsEnabled:  &animationsEnabled,
		LoaderInterval:     17 * time.Millisecond,
		Marks:              &marks,
		Getenv:             getenvFrom(nil),
		IsTerminal:         func(int) bool { return true },
	})

	colorEnabled = false
	debugEnabled = false
	interactiveEnabled = false
	unicodeEnabled = false
	animationsEnabled = false
	marks.Action = "changed"
	marks.SpinnerFrames[0] = "changed"

	if !console.SupportsColor() {
		t.Fatal("SupportsColor() = false after caller mutation, want true")
	}
	if !console.isDebugEnabled() {
		t.Fatal("isDebugEnabled() = false after caller mutation, want true")
	}
	if !console.IsInteractive() {
		t.Fatal("IsInteractive() = false after caller mutation, want true")
	}
	if !console.SupportsUnicode() {
		t.Fatal("SupportsUnicode() = false after caller mutation, want true")
	}
	if !console.shouldAnimate() {
		t.Fatal("shouldAnimate() = false after caller mutation, want true")
	}
	if got := console.loaderInterval; got != 17*time.Millisecond {
		t.Fatalf("loaderInterval = %s, want %s", got, 17*time.Millisecond)
	}
	if got := console.marks.Action; got != "action" {
		t.Fatalf("marks.Action = %q after caller mutation, want %q", got, "action")
	}
	if got := console.marks.SpinnerFrames[0]; got != "one" {
		t.Fatalf("marks.SpinnerFrames[0] = %q after caller mutation, want %q", got, "one")
	}
}

// TestNewSelectsFallbackRuntimeValues verifies conservative marks and loader timing when overrides are absent.
func TestNewSelectsFallbackRuntimeValues(t *testing.T) {
	t.Parallel()

	console := New(Config{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Getenv: getenvFrom(map[string]string{"TERM": "dumb"}),
	})

	if console.SupportsUnicode() {
		t.Fatal("SupportsUnicode() = true for TERM=dumb, want false")
	}
	if got := console.marks; !reflect.DeepEqual(got, ASCIIMarks()) {
		t.Fatalf("marks = %#v for TERM=dumb, want %#v", got, ASCIIMarks())
	}
	if got := console.loaderInterval; got != defaultLoaderInterval {
		t.Fatalf("loaderInterval = %s, want %s", got, defaultLoaderInterval)
	}
	if got := console.Width(); got != defaultTerminalWidth {
		t.Fatalf("Width() = %d, want %d", got, defaultTerminalWidth)
	}
}

// TestUnicodeDetectionAppliesOverrideAndLocalePrecedence verifies deterministic Unicode capability selection.
func TestUnicodeDetectionAppliesOverrideAndLocalePrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		override *bool
		env      map[string]string
		want     bool
	}{
		{name: "explicit enabled beats dumb terminal", override: boolPointer(true), env: map[string]string{"TERM": "dumb"}, want: true},
		{name: "explicit disabled beats UTF-8 locale", override: boolPointer(false), env: map[string]string{"LANG": "en_US.UTF-8"}, want: false},
		{name: "dumb terminal", env: map[string]string{"TERM": " DUMB ", "LANG": "en_US.UTF-8"}, want: false},
		{name: "LC_ALL C", env: map[string]string{"LC_ALL": " C ", "LANG": "en_US.UTF-8"}, want: false},
		{name: "LC_CTYPE POSIX", env: map[string]string{"LC_CTYPE": "posix"}, want: false},
		{name: "LANG C", env: map[string]string{"LANG": "C"}, want: false},
		{name: "hyphenated UTF-8", env: map[string]string{"LANG": "en_US.UTF-8"}, want: true},
		{name: "compact UTF8", env: map[string]string{"LC_CTYPE": "en_US.utf8"}, want: true},
		{name: "higher precedence non-UTF locale", env: map[string]string{"LC_ALL": "en_US.ISO-8859-1", "LANG": "en_US.UTF-8"}, want: false},
		{name: "unknown locale is conservative", env: map[string]string{"LANG": "english"}, want: false},
		{name: "empty environment defaults enabled", env: map[string]string{}, want: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := detectUnicode(test.override, getenvFrom(test.env)); got != test.want {
				t.Fatalf("detectUnicode() = %t, want %t", got, test.want)
			}
		})
	}
}

// TestColorDetectionAppliesOverrideEnvironmentAndTerminalPrecedence verifies common CLI color policy ordering.
func TestColorDetectionAppliesOverrideEnvironmentAndTerminalPrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		override   *bool
		env        map[string]string
		descriptor bool
		terminal   bool
		want       bool
	}{
		{name: "explicit enabled beats NO_COLOR", override: boolPointer(true), env: map[string]string{"NO_COLOR": "1", "TERM": "dumb"}, want: true},
		{name: "explicit disabled beats forced color", override: boolPointer(false), env: map[string]string{"CLICOLOR_FORCE": "1"}, descriptor: true, terminal: true, want: false},
		{name: "NO_COLOR beats forced color", env: map[string]string{"NO_COLOR": "1", "CLICOLOR_FORCE": "1"}, descriptor: true, terminal: true, want: false},
		{name: "forced color needs no descriptor", env: map[string]string{"CLICOLOR_FORCE": "1"}, want: true},
		{name: "nonzero forced color is enabled", env: map[string]string{"CLICOLOR_FORCE": "false"}, want: true},
		{name: "zero forced color falls through", env: map[string]string{"CLICOLOR_FORCE": "0"}, descriptor: true, terminal: true, want: true},
		{name: "CLICOLOR zero beats terminal", env: map[string]string{"CLICOLOR": "0"}, descriptor: true, terminal: true, want: false},
		{name: "dumb terminal beats descriptor", env: map[string]string{"TERM": "DUMB"}, descriptor: true, terminal: true, want: false},
		{name: "terminal descriptor", descriptor: true, terminal: true, want: true},
		{name: "redirected descriptor", descriptor: true, terminal: false, want: false},
		{name: "writer without descriptor", terminal: true, want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stdout io.Writer = &bytes.Buffer{}
			if test.descriptor {
				stdout = &descriptorBuffer{descriptor: 23}
			}
			console := New(Config{
				Stdout:       stdout,
				ColorEnabled: test.override,
				Getenv:       getenvFrom(test.env),
				IsTerminal: func(descriptor int) bool {
					if descriptor != 23 {
						t.Fatalf("IsTerminal() descriptor = %d, want 23", descriptor)
					}
					return test.terminal
				},
			})

			if got := console.SupportsColor(); got != test.want {
				t.Fatalf("SupportsColor() = %t, want %t", got, test.want)
			}
		})
	}
}

// TestEnvironmentFlagUsesConventionalCLITruthiness verifies that only empty and zero values disable a flag.
func TestEnvironmentFlagUsesConventionalCLITruthiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  bool
	}{
		{value: "", want: false},
		{value: "0", want: false},
		{value: "1", want: true},
		{value: "false", want: true},
		{value: " 0 ", want: true},
		{value: "yes", want: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()

			if got := environmentFlag(test.value); got != test.want {
				t.Fatalf("environmentFlag(%q) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}

// TestConsoleWidthUsesConfigurationDescriptorEnvironmentAndFallback verifies width source ordering and validation.
func TestConsoleWidthUsesConfigurationDescriptorEnvironmentAndFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configured    int
		descriptor    bool
		detected      int
		detectionErr  bool
		columns       string
		want          int
		wantSizeCalls int
	}{
		{name: "configured width", configured: 44, descriptor: true, detected: 120, columns: "99", want: 44, wantSizeCalls: 0},
		{name: "descriptor width", descriptor: true, detected: 120, columns: "99", want: 120, wantSizeCalls: 1},
		{name: "zero descriptor width uses columns", descriptor: true, detected: 0, columns: " 91 ", want: 91, wantSizeCalls: 1},
		{name: "negative descriptor width uses columns", descriptor: true, detected: -5, columns: "92", want: 92, wantSizeCalls: 1},
		{name: "descriptor error uses columns", descriptor: true, detectionErr: true, columns: "93", want: 93, wantSizeCalls: 1},
		{name: "writer without descriptor uses columns", columns: "94", want: 94, wantSizeCalls: 0},
		{name: "invalid columns uses fallback", columns: "wide", want: 80, wantSizeCalls: 0},
		{name: "zero columns uses fallback", columns: "0", want: 80, wantSizeCalls: 0},
		{name: "negative columns uses fallback", columns: "-1", want: 80, wantSizeCalls: 0},
		{name: "empty columns uses fallback", want: 80, wantSizeCalls: 0},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stdout io.Writer = &bytes.Buffer{}
			if test.descriptor {
				stdout = &descriptorBuffer{descriptor: 29}
			}
			sizeCalls := 0
			console := New(Config{
				Stdout: stdout,
				Width:  test.configured,
				Getenv: getenvFrom(map[string]string{"COLUMNS": test.columns}),
				GetSize: func(descriptor int) (int, int, error) {
					sizeCalls++
					if descriptor != 29 {
						t.Fatalf("GetSize() descriptor = %d, want 29", descriptor)
					}
					if test.detectionErr {
						return 0, 0, errors.New("size unavailable")
					}
					return test.detected, 40, nil
				},
			})

			if got := console.Width(); got != test.want {
				t.Fatalf("Width() = %d, want %d", got, test.want)
			}
			if sizeCalls != test.wantSizeCalls {
				t.Fatalf("GetSize() calls = %d, want %d", sizeCalls, test.wantSizeCalls)
			}
		})
	}
}

// TestInteractiveDetectionRequiresTerminalInputAndOutput verifies override and descriptor behavior.
func TestInteractiveDetectionRequiresTerminalInputAndOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		override         *bool
		inputDescriptor  bool
		outputDescriptor bool
		inputTerminal    bool
		outputTerminal   bool
		want             bool
	}{
		{name: "explicit enabled", override: boolPointer(true), want: true},
		{name: "explicit disabled", override: boolPointer(false), inputDescriptor: true, outputDescriptor: true, inputTerminal: true, outputTerminal: true, want: false},
		{name: "both terminals", inputDescriptor: true, outputDescriptor: true, inputTerminal: true, outputTerminal: true, want: true},
		{name: "redirected input", inputDescriptor: true, outputDescriptor: true, outputTerminal: true, want: false},
		{name: "redirected output", inputDescriptor: true, outputDescriptor: true, inputTerminal: true, want: false},
		{name: "input without descriptor", outputDescriptor: true, inputTerminal: true, outputTerminal: true, want: false},
		{name: "output without descriptor", inputDescriptor: true, inputTerminal: true, outputTerminal: true, want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stdin io.Reader = strings.NewReader("")
			if test.inputDescriptor {
				stdin = &descriptorReader{Reader: strings.NewReader(""), descriptor: 31}
			}
			var stdout io.Writer = &bytes.Buffer{}
			if test.outputDescriptor {
				stdout = &descriptorBuffer{descriptor: 32}
			}
			console := New(Config{
				Stdin:              stdin,
				Stdout:             stdout,
				InteractiveEnabled: test.override,
				IsTerminal: func(descriptor int) bool {
					switch descriptor {
					case 31:
						return test.inputTerminal
					case 32:
						return test.outputTerminal
					default:
						t.Fatalf("IsTerminal() descriptor = %d, want 31 or 32", descriptor)
						return false
					}
				},
			})

			if got := console.IsInteractive(); got != test.want {
				t.Fatalf("IsInteractive() = %t, want %t", got, test.want)
			}
		})
	}
}

// TestAnimationDetectionRequiresTerminalOutput verifies that an override cannot animate redirected output.
func TestAnimationDetectionRequiresTerminalOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		override   *bool
		env        map[string]string
		descriptor bool
		terminal   bool
		want       bool
	}{
		{name: "automatic terminal", descriptor: true, terminal: true, want: true},
		{name: "automatic dumb terminal", env: map[string]string{"TERM": " DUMB "}, descriptor: true, terminal: true, want: false},
		{name: "automatic redirected descriptor", descriptor: true, want: false},
		{name: "automatic writer without descriptor", terminal: true, want: false},
		{name: "explicit disabled terminal", override: boolPointer(false), descriptor: true, terminal: true, want: false},
		{name: "explicit enabled terminal", override: boolPointer(true), descriptor: true, terminal: true, want: true},
		{name: "explicit enabled dumb terminal", override: boolPointer(true), env: map[string]string{"TERM": "dumb"}, descriptor: true, terminal: true, want: true},
		{name: "explicit enabled redirected descriptor", override: boolPointer(true), descriptor: true, want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stdout io.Writer = &bytes.Buffer{}
			if test.descriptor {
				stdout = &descriptorBuffer{descriptor: 41}
			}
			console := New(Config{
				Stdout:            stdout,
				AnimationsEnabled: test.override,
				Getenv:            getenvFrom(test.env),
				IsTerminal: func(descriptor int) bool {
					if descriptor != 41 {
						t.Fatalf("IsTerminal() descriptor = %d, want 41", descriptor)
					}
					return test.terminal
				},
			})

			if got := console.shouldAnimate(); got != test.want {
				t.Fatalf("shouldAnimate() = %t, want %t", got, test.want)
			}
		})
	}
}

// TestDescriptorHelpersAcceptOnlyDescriptorImplementations verifies safe descriptor extraction.
func TestDescriptorHelpersAcceptOnlyDescriptorImplementations(t *testing.T) {
	t.Parallel()

	writer := &descriptorBuffer{descriptor: 51}
	if got, ok := writerDescriptor(writer); !ok || got != 51 {
		t.Fatalf("writerDescriptor() = (%d, %t), want (51, true)", got, ok)
	}
	if got, ok := writerDescriptor(&bytes.Buffer{}); ok || got != 0 {
		t.Fatalf("writerDescriptor() without Fd = (%d, %t), want (0, false)", got, ok)
	}

	reader := &descriptorReader{Reader: strings.NewReader(""), descriptor: 52}
	if got, ok := readerDescriptor(reader); !ok || got != 52 {
		t.Fatalf("readerDescriptor() = (%d, %t), want (52, true)", got, ok)
	}
	if got, ok := readerDescriptor(strings.NewReader("")); ok || got != 0 {
		t.Fatalf("readerDescriptor() without Fd = (%d, %t), want (0, false)", got, ok)
	}
}

// TestSetDefaultRejectsNil verifies that package helpers can never be left without a runtime.
func TestSetDefaultRejectsNil(t *testing.T) {
	defer func() {
		value := recover()
		if value != "console: default console cannot be nil" {
			t.Fatalf("SetDefault(nil) panic = %#v, want %q", value, "console: default console cannot be nil")
		}
	}()

	SetDefault(nil)
}
