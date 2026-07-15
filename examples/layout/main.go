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
	cli := console.New(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
		Width:          48,
	})

	cli.Section("Deployment")
	cli.KeyValues(
		console.KV("Environment", "production"),
		console.KV("Region", "eu-west-1"),
	)
	cli.Box(
		"The API and worker are healthy.",
		console.BoxTitle("Status"),
		console.BoxWidth(38),
	)
	cli.Table(
		[]string{"Service", "State"},
		[][]string{{"api", "ready"}, {"worker", "ready"}},
	)
}
