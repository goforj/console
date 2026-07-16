package main

import (
	"bytes"
	"testing"
)

// TestRun verifies the complete runnable deployment recipe.
func TestRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run(&stdout, &stderr)

	want := "◇ Deploy production\n" +
		"Environment  production\n" +
		"Region       eu-west-1\n" +
		"· Deploying services\n" +
		"✔ Services deployed\n" +
		"┌─────────┬───────┐\n" +
		"│ Service │ State │\n" +
		"├─────────┼───────┤\n" +
		"│ api     │ ready │\n" +
		"│ worker  │ ready │\n" +
		"└─────────┴───────┘\n" +
		"✔ Deployment complete\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant:\n%s", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
