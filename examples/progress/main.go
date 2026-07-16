// Command progress demonstrates the durable progress contract used for redirected output.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main advances determinate work and records an explicit successful outcome.
func main() {
	run(os.Stdout, os.Stderr)
}

// run writes the progress lifecycle to injected streams so its durable output is testable.
func run(stdout, stderr io.Writer) {
	color := false
	animations := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:            stdout,
		Stderr:            stderr,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	}))

	progress := console.NewProgress(100, "Packaging release")
	if err := progress.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Packaging release
	defer progress.Stop()
	progress.Step(40, "Uploading release")
	progress.Add(60)
	progress.Complete("Release ready")
	// ✔ Release ready
}
