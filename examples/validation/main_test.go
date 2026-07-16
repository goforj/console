package main

import (
	"bytes"
	"testing"
)

// TestRun verifies validation details stay on stdout while the terminal failure uses stderr.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	wantStdout := "◇ Configuration check\n" +
		"Checks  8\n" +
		"Passed  6\n" +
		"Failed  2\n" +
		"! 2 issues need attention\n" +
		"• DATABASE_URL is missing\n" +
		"• PORT must be between 1 and 65535\n"
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("stdout =\n%s\nwant:\n%s", got, wantStdout)
	}
	wantStderr := "✖ Validation failed\n"
	if got := stderr.String(); got != wantStderr {
		t.Fatalf("stderr = %q, want %q", got, wantStderr)
	}
}
