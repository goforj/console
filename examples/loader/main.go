// Command loader demonstrates the stable output used when animation is disabled or redirected.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main runs one loader lifecycle without terminal control sequences.
func main() {
	run(os.Stdout, os.Stderr)
}

// run writes the loader lifecycle to injected streams so its durable output is testable.
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

	loader := console.NewLoader("Downloading modules")
	if err := loader.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Downloading modules
	defer loader.Stop()
	loader.Update("Verifying modules")
	loader.Success("Modules ready")
	// ✔ Modules ready
}
