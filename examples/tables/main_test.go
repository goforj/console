package main

import (
	"bytes"
	"testing"
)

// TestRun verifies bordered, compact, and ASCII table output.
func TestRun(t *testing.T) {
	var output bytes.Buffer
	run(&output)

	want := "┌─────────┬───────┐\n" +
		"│ Service │ State │\n" +
		"├─────────┼───────┤\n" +
		"│ api     │ ready │\n" +
		"│ worker  │ ready │\n" +
		"└─────────┴───────┘\n" +
		"Task      Seconds\n" +
		"────────  ───────\n" +
		"compile        12\n" +
		"packages\n" +
		"test            3\n" +
		"+----------+-------+\n" +
		"|  Status  | Count |\n" +
		"+----------+-------+\n" +
		"|  ready   |     2 |\n" +
		"| waiting  |    12 |\n" +
		"+----------+-------+\n"
	if got := output.String(); got != want {
		t.Fatalf("output =\n%s\nwant:\n%s", got, want)
	}
}
