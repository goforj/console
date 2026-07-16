// Command validation demonstrates an actionable configuration report.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main renders a validation report with separate error output.
func main() {
	run(os.Stdout, os.Stderr)
}

// run writes the validation report to injected streams for exact verification.
func run(stdout, stderr io.Writer) {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         stdout,
		Stderr:         stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

	console.Section("Configuration check")
	// ◇ Configuration check
	console.KeyValues(
		console.KV("Checks", 8),
		console.KV("Passed", 6),
		console.KV("Failed", 2),
	)
	// Checks  8
	// Passed  6
	// Failed  2
	console.Warn("2 issues need attention")
	// ! 2 issues need attention
	console.List("DATABASE_URL is missing", "PORT must be between 1 and 65535")
	// • DATABASE_URL is missing
	// • PORT must be between 1 and 65535
	console.Error("Validation failed")
	// stderr: ✖ Validation failed
}
