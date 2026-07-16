// Command messages demonstrates semantic output through package-level helpers.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main prints the same semantic messages used by a typical command lifecycle.
func main() {
	run(os.Stdout, os.Stderr)
}

// run writes the example to injected streams so its documented output stays testable.
func run(stdout, stderr io.Writer) {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         stdout,
		Stderr:         stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

	console.Action("Building application")
	// · Building application
	console.Infof("Environment: %s", "development")
	// · Environment: development
	console.Success("Application ready")
	// ✔ Application ready
}
