// Command layout demonstrates composable console layout helpers.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main renders a compact deployment summary.
func main() {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
		Width:          48,
	}))

	console.Section("Deployment")
	// ◇ Deployment
	console.KeyValues(
		console.KV("Environment", "production"),
		console.KV("Region", "eu-west-1"),
	)
	// Environment  production
	// Region       eu-west-1
	console.Box(
		"The API and worker are healthy.",
		console.BoxTitle("Status"),
		console.BoxWidth(38),
	)
	// ┌─ Status ───────────────────────────┐
	// │ The API and worker are healthy.    │
	// └────────────────────────────────────┘
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
}
