// Command progress demonstrates the durable progress contract used for redirected output.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main advances determinate work and records an explicit successful outcome.
func main() {
	color := false
	animations := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:            os.Stdout,
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
	progress.Set(40)
	progress.Update("Uploading release")
	progress.Add(60)
	progress.Complete("Release ready")
	// ✔ Release ready
}
