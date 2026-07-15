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
		{name: "confirm", call: func() error { _, err := console.Confirm("Continue", true); return err }},
		{name: "choose", call: func() error { _, err := console.Choose("Pick", []string{"one"}, 0); return err }},
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

	configured, _, _ := newPromptTestConsole(strings.NewReader("Ada\n\nno\n2\n"), true)
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
}
