// Command messages demonstrates semantic output through package-level helpers.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main prints the same semantic messages used by a typical command lifecycle.
func main() {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

	console.Action("Building application")
	console.Infof("Environment: %s", "development")
	console.Success("Application ready")

	// · Building application
	// · Environment: development
	// ✔ Application ready
}
