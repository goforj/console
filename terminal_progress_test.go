package console

import (
	"bytes"
	"strings"
	"testing"
)

// newTerminalProgressTestConsole creates a static ASCII console whose terminal capabilities are deterministic.
func newTerminalProgressTestConsole(enabled *bool, terminal bool) (*Console, *loaderTestWriter, *loaderTestWriter) {
	animations := false
	color := false
	unicode := false
	stdout := newLoaderTestWriter(1)
	stderr := newLoaderTestWriter(2)
	commandConsole := New(Config{
		Stdout:                  stdout,
		Stderr:                  stderr,
		AnimationsEnabled:       &animations,
		ColorEnabled:            &color,
		UnicodeEnabled:          &unicode,
		TerminalProgressEnabled: enabled,
		IsTerminal:              func(int) bool { return terminal },
	})
	return commandConsole, stdout, stderr
}

func TestLoaderPublishesIndeterminateTerminalProgress(t *testing.T) {
	enabled := true
	commandConsole, stdout, stderr := newTerminalProgressTestConsole(&enabled, true)
	loader := commandConsole.Loader("Loading project")

	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	loader.Success("Project ready")
	loader.Fail("ignored")

	want := "- Loading project\n" +
		terminalProgressSequence(terminalProgressStateIndeterminate, 0) +
		terminalProgressSequence(terminalProgressStateClear, 0) +
		"+ Project ready\n"
	if got := stdout.String(); got != want {
		t.Fatalf("loader output = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestProgressPublishesDeterminateTerminalProgress(t *testing.T) {
	enabled := true
	commandConsole, stdout, _ := newTerminalProgressTestConsole(&enabled, true)
	progress := commandConsole.Progress(4, "Copying files")
	progress.Set(1)

	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	progress.Add(1)
	progress.Step(3, "Installing files")
	progress.Complete("Files installed")
	progress.Set(1)

	want := "- Copying files\n" +
		terminalProgressSequence(terminalProgressStateDeterminate, 25) +
		terminalProgressSequence(terminalProgressStateDeterminate, 50) +
		terminalProgressSequence(terminalProgressStateDeterminate, 75) +
		terminalProgressSequence(terminalProgressStateClear, 0) +
		"+ Files installed\n"
	if got := stdout.String(); got != want {
		t.Fatalf("progress output = %q, want %q", got, want)
	}
}

func TestTerminalProgressRequiresExplicitTerminalCapability(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		terminal bool
	}{
		{name: "unset", terminal: true},
		{name: "disabled", enabled: boolPointer(false), terminal: true},
		{name: "redirected", enabled: boolPointer(true)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			commandConsole, stdout, _ := newTerminalProgressTestConsole(test.enabled, test.terminal)
			loader := commandConsole.Loader("work")
			if err := loader.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			loader.Stop()
			if strings.Contains(stdout.String(), "\x1b]9;4;") {
				t.Fatalf("output contains terminal progress: %q", stdout.String())
			}
		})
	}

	t.Run("writer without descriptor", func(t *testing.T) {
		enabled := true
		output := &bytes.Buffer{}
		commandConsole := New(Config{
			Stdout:                  output,
			TerminalProgressEnabled: &enabled,
			IsTerminal:              func(int) bool { return true },
		})
		loader := commandConsole.Loader("work")
		if err := loader.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		loader.Stop()
		if strings.Contains(output.String(), "\x1b]9;4;") {
			t.Fatalf("output contains terminal progress: %q", output.String())
		}
	})

	t.Run("terminal without ANSI support", func(t *testing.T) {
		enabled := true
		commandConsole, stdout, _ := newTerminalProgressTestConsole(&enabled, true)
		commandConsole.supportsANSI = func(int) bool { return false }
		loader := commandConsole.Loader("work")
		if err := loader.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		loader.Stop()
		if strings.Contains(stdout.String(), "\x1b]9;4;") {
			t.Fatalf("output contains terminal progress: %q", stdout.String())
		}
	})
}

func TestTerminalProgressOwnershipPreventsCrossLifecycleClears(t *testing.T) {
	enabled := true
	commandConsole, stdout, _ := newTerminalProgressTestConsole(&enabled, true)
	first := commandConsole.Loader("first")
	second := commandConsole.Loader("second")

	if err := first.Start(); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	if err := second.Start(); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	second.Stop()
	first.Stop()
	commandConsole.setTerminalProgress(first, terminalProgressStateIndeterminate, 0)

	got := stdout.String()
	if count := strings.Count(got, terminalProgressSequence(terminalProgressStateIndeterminate, 0)); count != 1 {
		t.Fatalf("indeterminate sequence count = %d, want 1: %q", count, got)
	}
	if count := strings.Count(got, terminalProgressSequence(terminalProgressStateClear, 0)); count != 1 {
		t.Fatalf("clear sequence count = %d, want 1: %q", count, got)
	}
}

func TestTerminalProgressSequenceClampsPercent(t *testing.T) {
	if got, want := terminalProgressSequence(terminalProgressStateDeterminate, -1), "\x1b]9;4;1;0\x07"; got != want {
		t.Fatalf("negative sequence = %q, want %q", got, want)
	}
	if got, want := terminalProgressSequence(terminalProgressStateDeterminate, 101), "\x1b]9;4;1;100\x07"; got != want {
		t.Fatalf("overflow sequence = %q, want %q", got, want)
	}
}
