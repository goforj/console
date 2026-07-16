package main

import (
	"bytes"
	"testing"
)

// TestRun verifies every rendered layout in the runnable example.
func TestRun(t *testing.T) {
	var output bytes.Buffer
	run(&output)

	want := "◇ Deployment\n" +
		"Environment  production\n" +
		"Region       eu-west-1\n" +
		"┌─ Status ───────────────────────────┐\n" +
		"│ The API and worker are healthy.    │\n" +
		"└────────────────────────────────────┘\n" +
		"┌─────────┬───────┐\n" +
		"│ Service │ State │\n" +
		"├─────────┼───────┤\n" +
		"│ api     │ ready │\n" +
		"│ worker  │ ready │\n" +
		"└─────────┴───────┘\n"
	if got := output.String(); got != want {
		t.Fatalf("output =\n%s\nwant:\n%s", got, want)
	}
}
