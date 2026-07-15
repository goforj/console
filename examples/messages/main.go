// Command messages demonstrates semantic output with an isolated console.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main prints the same semantic messages used by a typical command lifecycle.
func main() {
	color := false
	unicode := true
	cli := console.New(console.Config{
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})

	cli.Action("Building application")
	cli.Infof("Environment: %s", "development")
	cli.Success("Application ready")
}
