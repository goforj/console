// Command tables demonstrates the default and compact table presentations.
package main

import (
	"os"

	"github.com/goforj/console"
)

// main renders one framed table followed by a compact, aligned summary.
func main() {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

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

	console.Table(
		[]string{"Task", "Seconds"},
		[][]string{{"compile packages", "12"}, {"test", "3"}},
		console.TableCompact(),
		console.TableWidths(8, 7),
		console.TableRightAlign(1),
	)
	// Task      Seconds
	// ────────  ───────
	// compile        12
	// packages
	// test            3
}
