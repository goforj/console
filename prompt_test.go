package console

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// promptReadSpy records whether a gated prompt attempted to consume input.
type promptReadSpy struct {
	reads int
}

// Read records the unexpected read and reports end of input.
func (r *promptReadSpy) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}

// promptErrorReader returns one stable injected input failure.
type promptErrorReader struct {
	err error
}

// Read returns the configured failure without producing input.
func (r *promptErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

// newPromptTestConsole creates a deterministic interactive or gated ASCII prompt console.
func newPromptTestConsole(input io.Reader, interactive bool) (*Console, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	colorEnabled := false
	unicodeEnabled := false
	return New(Config{
		Stdin:              input,
		Stdout:             stdout,
		Stderr:             stderr,
		ColorEnabled:       &colorEnabled,
		InteractiveEnabled: &interactive,
		UnicodeEnabled:     &unicodeEnabled,
		Width:              40,
	}), stdout, stderr
}

// TestPromptMethodsRejectNonInteractiveInputWithoutReading verifies gating precedes output and input access.
func TestPromptMethodsRejectNonInteractiveInputWithoutReading(t *testing.T) {
	reader := &promptReadSpy{}
	console, stdout, stderr := newPromptTestConsole(reader, false)

	tests := []struct {
		name string
		call func() error
	}{
		{name: "ask", call: func() error { _, err := console.Ask("Name"); return err }},
		{name: "ask default", call: func() error { _, err := console.AskDefault("Name", "Ada"); return err }},
		{name: "ask secret", call: func() error { _, err := console.AskSecret("Password"); return err }},
		{name: "confirm", call: func() error { _, err := console.Confirm("Continue", true); return err }},
		{name: "choose", call: func() error { _, err := console.Choose("Pick", []string{"one"}, 0); return err }},
		{name: "choose index", call: func() error { _, err := console.ChooseIndex("Pick", []string{"one"}, 0); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, ErrNonInteractive) {
				t.Fatalf("prompt error = %v, want ErrNonInteractive", err)
			}
		})
	}

	if reader.reads != 0 {
		t.Fatalf("input reads = %d, want 0", reader.reads)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestPromptMethodsPropagateInitialOutputErrors verifies input is never consumed after a prompt cannot be shown.
func TestPromptMethodsPropagateInitialOutputErrors(t *testing.T) {
	tests := []struct {
		name string
		call func(*Console) error
	}{
		{name: "ask", call: func(console *Console) error { _, err := console.Ask("Name"); return err }},
		{name: "ask default", call: func(console *Console) error { _, err := console.AskDefault("Name", "Ada"); return err }},
		{name: "ask secret", call: func(console *Console) error { _, err := console.AskSecret("Password"); return err }},
		{name: "confirm", call: func(console *Console) error { _, err := console.Confirm("Continue", true); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantErr := errors.New("prompt output failed")
			reader := &promptReadSpy{}
			secretCalls := 0
			interactive := true
			console := New(Config{
				Stdin:              reader,
				Stdout:             failingOutputWriter{err: wantErr},
				ColorEnabled:       boolPointer(false),
				InteractiveEnabled: &interactive,
				UnicodeEnabled:     boolPointer(false),
				Getenv:             getenvFrom(nil),
				ReadSecret: func() (string, error) {
					secretCalls++
					return "secret", nil
				},
			})

			if err := test.call(console); !errors.Is(err, wantErr) {
				t.Fatalf("prompt error = %v, want %v", err, wantErr)
			}
			if reader.reads != 0 {
				t.Fatalf("input reads = %d, want 0", reader.reads)
			}
			if secretCalls != 0 {
				t.Fatalf("secret reads = %d, want 0", secretCalls)
			}
			if console.promptActive || console.partialLine {
				t.Fatalf("prompt state = (active %t, partial %t), want cleared", console.promptActive, console.partialLine)
			}
		})
	}
}

// TestChoiceMethodsPropagatePreludeOutputErrors verifies choice input waits until every option is visible.
func TestChoiceMethodsPropagatePreludeOutputErrors(t *testing.T) {
	wantErr := errors.New("choice output failed")
	reader := &promptReadSpy{}
	interactive := true
	console := New(Config{
		Stdin:              reader,
		Stdout:             failingOutputWriter{err: wantErr},
		ColorEnabled:       boolPointer(false),
		InteractiveEnabled: &interactive,
		UnicodeEnabled:     boolPointer(false),
		Getenv:             getenvFrom(nil),
	})

	choice, err := console.Choose("Pick", []string{"one"}, -1)
	if choice != "" || !errors.Is(err, wantErr) {
		t.Fatalf("Choose() = (%q, %v), want (empty, %v)", choice, err, wantErr)
	}
	index, err := console.ChooseIndex("Pick", []string{"one"}, -1)
	if index != -1 || !errors.Is(err, wantErr) {
		t.Fatalf("ChooseIndex() = (%d, %v), want (-1, %v)", index, err, wantErr)
	}
	if reader.reads != 0 {
		t.Fatalf("input reads = %d, want 0", reader.reads)
	}
}

// TestPromptPartialOutputFailureCompletesLineAndRestoresTransient verifies failed prompts leave reusable state.
func TestPromptPartialOutputFailureCompletesLineAndRestoresTransient(t *testing.T) {
	wantErr := errors.New("partial prompt failed")
	stdout := &scriptedOutputWriter{results: []scriptedWriteResult{
		{accepted: -1},
		{accepted: 2, err: wantErr},
		{accepted: -1},
		{accepted: -1},
	}}
	reader := &promptReadSpy{}
	interactive := true
	console := New(Config{
		Stdin:              reader,
		Stdout:             stdout,
		ColorEnabled:       boolPointer(false),
		InteractiveEnabled: &interactive,
		UnicodeEnabled:     boolPointer(false),
		Getenv:             getenvFrom(nil),
	})
	console.active = staticTransient("frame")

	value, err := console.Ask("Name")
	if value != "" || !errors.Is(err, wantErr) {
		t.Fatalf("Ask() = (%q, %v), want (empty, %v)", value, err, wantErr)
	}
	if reader.reads != 0 {
		t.Fatalf("input reads = %d, want 0", reader.reads)
	}
	if console.promptActive || console.partialLine {
		t.Fatalf("prompt state = (active %t, partial %t), want cleared", console.promptActive, console.partialLine)
	}
	if got, want := stdout.String(), clearTransientLine+"> \nframe"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestChoicePartialPreludeFailureCompletesLineAndRestoresTransient verifies list failures stop before input.
func TestChoicePartialPreludeFailureCompletesLineAndRestoresTransient(t *testing.T) {
	wantErr := errors.New("partial choice failed")
	stdout := &scriptedOutputWriter{results: []scriptedWriteResult{
		{accepted: -1},
		{accepted: 4, err: wantErr},
		{accepted: -1},
		{accepted: -1},
	}}
	reader := &promptReadSpy{}
	interactive := true
	console := New(Config{
		Stdin:              reader,
		Stdout:             stdout,
		ColorEnabled:       boolPointer(false),
		InteractiveEnabled: &interactive,
		UnicodeEnabled:     boolPointer(false),
		Getenv:             getenvFrom(nil),
	})
	console.active = staticTransient("frame")

	index, err := console.ChooseIndex("Pick", []string{"one"}, -1)
	if index != -1 || !errors.Is(err, wantErr) {
		t.Fatalf("ChooseIndex() = (%d, %v), want (-1, %v)", index, err, wantErr)
	}
	if reader.reads != 0 {
		t.Fatalf("input reads = %d, want 0", reader.reads)
	}
	if console.promptActive || console.partialLine {
		t.Fatalf("prompt state = (active %t, partial %t), want cleared", console.promptActive, console.partialLine)
	}
	if got, want := stdout.String(), clearTransientLine+"Pick\nframe"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestPromptReturnsInputAndTransientCleanupErrors verifies coordinated redraw failures are not discarded.
func TestPromptReturnsInputAndTransientCleanupErrors(t *testing.T) {
	inputErr := errors.New("input failed")
	redrawErr := errors.New("redraw failed")
	stdout := &scriptedOutputWriter{results: []scriptedWriteResult{
		{accepted: -1},
		{accepted: -1},
		{accepted: -1},
		{accepted: 0, err: redrawErr},
	}}
	interactive := true
	console := New(Config{
		Stdin:              &promptErrorReader{err: inputErr},
		Stdout:             stdout,
		ColorEnabled:       boolPointer(false),
		InteractiveEnabled: &interactive,
		UnicodeEnabled:     boolPointer(false),
		Getenv:             getenvFrom(nil),
	})
	console.active = staticTransient("frame")

	_, err := console.Ask("Name")
	if !errors.Is(err, inputErr) || !errors.Is(err, redrawErr) {
		t.Fatalf("Ask() error = %v, want joined input and redraw errors", err)
	}
	if console.promptActive || console.partialLine {
		t.Fatalf("prompt state = (active %t, partial %t), want cleared", console.promptActive, console.partialLine)
	}
	if got, want := stdout.String(), clearTransientLine+"> Name: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestPromptRetryWarningsPropagateOutputErrors verifies invalid input does not hide failed guidance.
func TestPromptRetryWarningsPropagateOutputErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		results    []scriptedWriteResult
		call       func(*Console) error
		wantOutput string
	}{
		{
			name:    "confirm",
			input:   "maybe\n",
			results: []scriptedWriteResult{{accepted: -1}, {accepted: 0, err: errors.New("warning failed")}},
			call: func(console *Console) error {
				_, err := console.Confirm("Continue", false)
				return err
			},
			wantOutput: "> Continue [y/N]: ",
		},
		{
			name:  "choose",
			input: "bad\n",
			results: []scriptedWriteResult{
				{accepted: -1},
				{accepted: -1},
				{accepted: 0, err: errors.New("warning failed")},
			},
			call: func(console *Console) error {
				_, err := console.ChooseIndex("Pick", []string{"one"}, -1)
				return err
			},
			wantOutput: "Pick\n1. one\n> Choose [1-1]: ",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout := &scriptedOutputWriter{results: test.results}
			interactive := true
			console := New(Config{
				Stdin:              strings.NewReader(test.input),
				Stdout:             stdout,
				ColorEnabled:       boolPointer(false),
				InteractiveEnabled: &interactive,
				UnicodeEnabled:     boolPointer(false),
				Getenv:             getenvFrom(nil),
			})
			if err := test.call(console); err == nil || !strings.Contains(err.Error(), "warning failed") {
				t.Fatalf("prompt error = %v, want warning output failure", err)
			}
			if got := stdout.String(); got != test.wantOutput {
				t.Fatalf("stdout = %q, want %q", got, test.wantOutput)
			}
		})
	}
}

// TestAskSecretUsesInjectedReader verifies secrets are returned exactly without appearing in output.
func TestAskSecretUsesInjectedReader(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	interactive := true
	color := false
	unicode := false
	calls := 0
	console := New(Config{
		Stdin:              strings.NewReader("ignored"),
		Stdout:             stdout,
		Stderr:             stderr,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
		ReadSecret: func() (string, error) {
			calls++
			return "  s3cr3t  ", nil
		},
	})

	secret, err := console.AskSecret("Password")
	if err != nil {
		t.Fatalf("AskSecret() error = %v", err)
	}
	if secret != "  s3cr3t  " {
		t.Fatalf("AskSecret() = %q, want exact secret", secret)
	}
	if calls != 1 {
		t.Fatalf("ReadSecret calls = %d, want 1", calls)
	}
	if got, want := stdout.String(), "> Password: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestAskSecretPropagatesReaderErrors verifies hidden-input failures remain available to callers.
func TestAskSecretPropagatesReaderErrors(t *testing.T) {
	wantErr := errors.New("secret input failed")
	stdout := &bytes.Buffer{}
	interactive := true
	color := false
	unicode := false
	console := New(Config{
		Stdin:              strings.NewReader(""),
		Stdout:             stdout,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
		ReadSecret: func() (string, error) {
			return "", wantErr
		},
	})

	secret, err := console.AskSecret("Token")
	if !errors.Is(err, wantErr) {
		t.Fatalf("AskSecret() error = %v, want %v", err, wantErr)
	}
	if secret != "" {
		t.Fatalf("AskSecret() = %q, want empty", secret)
	}
	if got, want := stdout.String(), "> Token: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestAskSecretRequiresTerminalReaderByDefault verifies explicit interactivity never weakens to echoed input.
func TestAskSecretRequiresTerminalReaderByDefault(t *testing.T) {
	console, stdout, _ := newPromptTestConsole(strings.NewReader("visible secret\n"), true)
	secret, err := console.AskSecret("Password")
	if err == nil || !strings.Contains(err.Error(), "terminal reader or Config.ReadSecret") {
		t.Fatalf("AskSecret() error = %v, want terminal-reader guidance", err)
	}
	if secret != "" {
		t.Fatalf("AskSecret() = %q, want empty", secret)
	}
	if got, want := stdout.String(), "> Password: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestPromptsReuseBufferedInputAcrossLFCRLFAndEOF verifies one reader survives sequential prompt methods.
func TestPromptsReuseBufferedInputAcrossLFCRLFAndEOF(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("Ada\nyes\r\n2"), true)

	name, err := console.Ask("Name")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if name != "Ada" {
		t.Fatalf("Ask() = %q, want %q", name, "Ada")
	}
	confirmed, err := console.Confirm("Continue", false)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if !confirmed {
		t.Fatal("Confirm() = false, want true")
	}
	choice, err := console.Choose("Pick one", []string{"red", "blue"}, -1)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if choice != "blue" {
		t.Fatalf("Choose() = %q, want %q", choice, "blue")
	}

	want := "> Name: > Continue [y/N]: Pick one\n1. red\n2. blue\n> Choose [1-2]: \n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestAskReturnsEOFWithoutInput verifies end of input cannot masquerade as an empty submitted line.
func TestAskReturnsEOFWithoutInput(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader(""), true)
	value, err := console.Ask("Name")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Ask() error = %v, want io.EOF", err)
	}
	if value != "" {
		t.Fatalf("Ask() = %q, want empty", value)
	}
	if got, want := stdout.String(), "> Name: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestAskDefaultDistinguishesSubmittedEmptyLinesFromEOF verifies defaults require an actual submitted line.
func TestAskDefaultDistinguishesSubmittedEmptyLinesFromEOF(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantEOF   bool
	}{
		{name: "LF", input: "\n", wantValue: "Ada"},
		{name: "CRLF and whitespace", input: " \t\r\n", wantValue: "Ada"},
		{name: "EOF", input: "", wantEOF: true},
		{name: "unterminated value", input: " Grace ", wantValue: "Grace"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			console, stdout, stderr := newPromptTestConsole(strings.NewReader(test.input), true)
			value, err := console.AskDefault("Name", "Ada")
			if test.wantEOF {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("AskDefault() error = %v, want io.EOF", err)
				}
			} else if err != nil {
				t.Fatalf("AskDefault() error = %v", err)
			}
			if value != test.wantValue {
				t.Fatalf("AskDefault() = %q, want %q", value, test.wantValue)
			}
			wantOutput := "> Name [Ada]: "
			if !strings.HasSuffix(test.input, "\n") {
				wantOutput += "\n"
			}
			if got := stdout.String(); got != wantOutput {
				t.Fatalf("stdout = %q, want %q", got, wantOutput)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
		})
	}
}

// TestAskPropagatesReaderErrors verifies prompt input failures remain available to callers.
func TestAskPropagatesReaderErrors(t *testing.T) {
	wantErr := errors.New("input failed")
	console, stdout, stderr := newPromptTestConsole(&promptErrorReader{err: wantErr}, true)
	value, err := console.Ask("Name")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Ask() error = %v, want %v", err, wantErr)
	}
	if value != "" {
		t.Fatalf("Ask() = %q, want empty", value)
	}
	if got, want := stdout.String(), "> Name: \n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestConfirmRetriesInvalidAnswersAndAcceptsCaseInsensitiveYes verifies retry output and answer normalization.
func TestConfirmRetriesInvalidAnswersAndAcceptsCaseInsensitiveYes(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("maybe\r\n YES \n"), true)
	confirmed, err := console.Confirm("Deploy", false)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if !confirmed {
		t.Fatal("Confirm() = false, want true")
	}
	want := "> Deploy [y/N]: ! Please answer yes or no.\n> Deploy [y/N]: "
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestConfirmHandlesDefaultsExplicitNoAndEOF verifies confirmation defaults never consume bare EOF.
func TestConfirmHandlesDefaultsExplicitNoAndEOF(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue bool
		want         bool
		wantEOF      bool
		wantHint     string
	}{
		{name: "true default", input: "\n", defaultValue: true, want: true, wantHint: "[Y/n]"},
		{name: "false default", input: "\r\n", defaultValue: false, want: false, wantHint: "[y/N]"},
		{name: "explicit no", input: "NO\n", defaultValue: true, want: false, wantHint: "[Y/n]"},
		{name: "EOF with true default", input: "", defaultValue: true, want: false, wantEOF: true, wantHint: "[Y/n]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			console, stdout, stderr := newPromptTestConsole(strings.NewReader(test.input), true)
			got, err := console.Confirm("Continue", test.defaultValue)
			if test.wantEOF {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("Confirm() error = %v, want io.EOF", err)
				}
			} else if err != nil {
				t.Fatalf("Confirm() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Confirm() = %t, want %t", got, test.want)
			}
			wantOutput := "> Continue " + test.wantHint + ": "
			if test.wantEOF {
				wantOutput += "\n"
			}
			if output := stdout.String(); output != wantOutput {
				t.Fatalf("stdout = %q, want %q", output, wantOutput)
			}
			if output := stderr.String(); output != "" {
				t.Fatalf("stderr = %q, want empty", output)
			}
		})
	}
}

// TestChooseValidatesOptionsBeforePrompting verifies invalid choice configuration performs no I/O.
func TestChooseValidatesOptionsBeforePrompting(t *testing.T) {
	tests := []struct {
		name         string
		options      []string
		defaultIndex int
		wantError    string
	}{
		{name: "empty options", options: nil, defaultIndex: -1, wantError: "at least one option"},
		{name: "default below range", options: []string{"one"}, defaultIndex: -2, wantError: "outside"},
		{name: "default above range", options: []string{"one"}, defaultIndex: 1, wantError: "outside"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &promptReadSpy{}
			console, stdout, stderr := newPromptTestConsole(reader, true)
			value, err := console.Choose("Pick", test.options, test.defaultIndex)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Choose() error = %v, want text %q", err, test.wantError)
			}
			if value != "" {
				t.Fatalf("Choose() = %q, want empty", value)
			}
			index, indexErr := console.ChooseIndex("Pick", test.options, test.defaultIndex)
			if indexErr == nil || !strings.Contains(indexErr.Error(), test.wantError) {
				t.Fatalf("ChooseIndex() error = %v, want text %q", indexErr, test.wantError)
			}
			if index != -1 {
				t.Fatalf("ChooseIndex() = %d, want -1", index)
			}
			if reader.reads != 0 {
				t.Fatalf("input reads = %d, want 0", reader.reads)
			}
			if got := stdout.String(); got != "" {
				t.Fatalf("stdout = %q, want empty", got)
			}
			if got := stderr.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
		})
	}
}

// TestChooseIndexReturnsZeroBasedSelection verifies callers can retain option identity without value lookup.
func TestChooseIndexReturnsZeroBasedSelection(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("2\n"), true)
	index, err := console.ChooseIndex("Pick one", []string{"red", "blue"}, -1)
	if err != nil {
		t.Fatalf("ChooseIndex() error = %v", err)
	}
	if index != 1 {
		t.Fatalf("ChooseIndex() = %d, want 1", index)
	}
	if got, want := stdout.String(), "Pick one\n1. red\n2. blue\n> Choose [1-2]: "; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestChooseRetriesInvalidSelectionsAndReturnsValue verifies range checks and stable repeated prompts.
func TestChooseRetriesInvalidSelectionsAndReturnsValue(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("0\nword\r\n3\n"), true)
	choice, err := console.Choose("Pick one", []string{"red", "green", "blue"}, 1)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if choice != "blue" {
		t.Fatalf("Choose() = %q, want %q", choice, "blue")
	}
	want := "Pick one\n1. red\n2. green\n3. blue\n" +
		"> Choose [1-3, default 2]: ! Choose a number from 1 to 3.\n" +
		"> Choose [1-3, default 2]: ! Choose a number from 1 to 3.\n" +
		"> Choose [1-3, default 2]: "
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestChooseUsesSubmittedEmptyLineForDefault verifies a default choice requires an actual empty line.
func TestChooseUsesSubmittedEmptyLineForDefault(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("\n"), true)
	choice, err := console.Choose("Pick one", []string{"red", "green"}, 1)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if choice != "green" {
		t.Fatalf("Choose() = %q, want %q", choice, "green")
	}
	want := "Pick one\n1. red\n2. green\n> Choose [1-2, default 2]: "
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestChooseWithoutDefaultRetriesEmptyInputThenReturnsEOF verifies required choices do not invent a selection.
func TestChooseWithoutDefaultRetriesEmptyInputThenReturnsEOF(t *testing.T) {
	console, stdout, stderr := newPromptTestConsole(strings.NewReader("\n"), true)
	choice, err := console.Choose("Pick one", []string{"red", "green"}, -1)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Choose() error = %v, want io.EOF", err)
	}
	if choice != "" {
		t.Fatalf("Choose() = %q, want empty", choice)
	}
	want := "Pick one\n1. red\n2. green\n" +
		"> Choose [1-2]: ! Choose a number from 1 to 2.\n" +
		"> Choose [1-2]: \n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestPackagePromptHelpersRouteThroughDefault verifies every prompt wrapper snapshots the configured default console.
func TestPackagePromptHelpersRouteThroughDefault(t *testing.T) {
	previous := Default()
	t.Cleanup(func() {
		SetDefault(previous)
	})

	configured, _, _ := newPromptTestConsole(strings.NewReader("Ada\n\nno\n2\n1\n"), true)
	configured.readSecret = func() (string, error) {
		return "token", nil
	}
	SetDefault(configured)

	name, err := Ask("Name")
	if err != nil || name != "Ada" {
		t.Fatalf("Ask() = %q, %v; want %q, nil", name, err, "Ada")
	}
	region, err := AskDefault("Region", "west")
	if err != nil || region != "west" {
		t.Fatalf("AskDefault() = %q, %v; want %q, nil", region, err, "west")
	}
	confirmed, err := Confirm("Continue", true)
	if err != nil || confirmed {
		t.Fatalf("Confirm() = %t, %v; want false, nil", confirmed, err)
	}
	choice, err := Choose("Pick", []string{"one", "two"}, -1)
	if err != nil || choice != "two" {
		t.Fatalf("Choose() = %q, %v; want %q, nil", choice, err, "two")
	}
	index, err := ChooseIndex("Pick index", []string{"one", "two"}, -1)
	if err != nil || index != 0 {
		t.Fatalf("ChooseIndex() = %d, %v; want 0, nil", index, err)
	}
	secret, err := AskSecret("Token")
	if err != nil || secret != "token" {
		t.Fatalf("AskSecret() = %q, %v; want %q, nil", secret, err, "token")
	}
}
