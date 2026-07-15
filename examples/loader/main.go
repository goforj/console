// Command loader demonstrates the stable output used when animation is disabled or redirected.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main runs one loader lifecycle without terminal control sequences.
func main() {
	color := false
	animations := false
	unicode := true
	cli := console.New(console.Config{
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	})

	loader := cli.Loader("Downloading modules")
	if err := loader.Start(); err != nil {
		cli.Error(err.Error())
		return
	}
	loader.Update("Verifying modules")
	loader.Success("Modules ready")
}
