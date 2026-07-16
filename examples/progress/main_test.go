package main

import (
	"bytes"
	"testing"
)

// TestRun verifies redirected progress output records only durable lifecycle events.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	want := "· Packaging release\n✔ Release ready\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
