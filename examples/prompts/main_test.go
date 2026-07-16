package main

import (
	"bytes"
	"testing"
)

// TestRun verifies the complete scripted prompt exchange and selected values.
func TestRun(t *testing.T) {
	var output bytes.Buffer
	run(&output)

	want := "› Name:\n" +
		"› Environment [production]:\n" +
		"› Deploy now [y/N]:\n" +
		"Ada production true\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
