package main

import (
	"bytes"
	"testing"
)

// TestRun verifies machine-readable stdout remains uncontaminated by status output.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	wantStdout := "{\"artifact\":\"app.tar.gz\",\"status\":\"ready\"}\n"
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("stdout = %q, want %q", got, wantStdout)
	}
	wantStderr := "status: uploading app.tar.gz\n"
	if got := stderr.String(); got != wantStderr {
		t.Fatalf("stderr = %q, want %q", got, wantStderr)
	}
}
