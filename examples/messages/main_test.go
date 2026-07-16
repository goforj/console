package main

import (
	"bytes"
	"testing"
)

// TestRun verifies the runnable message example and its adjacent output comments remain exact.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	want := "· Building application\n" +
		"· Environment: development\n" +
		"✔ Application ready\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
