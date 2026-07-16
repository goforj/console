// Command deploy demonstrates a complete deployment console experience with package helpers.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main renders the complete deployment lifecycle.
func main() {
	run(os.Stdout, os.Stderr)
}

// run writes a deterministic deployment lifecycle to injected streams.
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

	console.Section("Deploy production")
	// ◇ Deploy production
	console.KeyValues(
		console.KV("Environment", "production"),
		console.KV("Region", "eu-west-1"),
	)
	// Environment  production
	// Region       eu-west-1

	progress := console.NewProgress(2, "Deploying services")
	if err := progress.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Deploying services
	defer progress.Stop()
	progress.Step(1, "Deploying worker")
	progress.Complete("Services deployed")
	// ✔ Services deployed

	console.Table(
		[]string{"Service", "State"},
		[][]string{{"api", "ready"}, {"worker", "ready"}},
	)
	// ┌─────────┬───────┐
	// │ Service │ State │
	// ├─────────┼───────┤
	// │ api     │ ready │
	// │ worker  │ ready │
	// └─────────┴───────┘
	console.Success("Deployment complete")
	// ✔ Deployment complete
}
