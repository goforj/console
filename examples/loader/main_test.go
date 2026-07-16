package main

import (
	"bytes"
	"testing"
)

// TestRun verifies redirected loader output stays durable and free of control sequences.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	want := "· Downloading modules\n✔ Modules ready\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
