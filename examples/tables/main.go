// Command tables demonstrates the default and compact table presentations.
package main

import (
	"io"
	"os"

	"github.com/goforj/console"
)

// main renders one framed table followed by a compact, aligned summary.
func main() {
	run(os.Stdout)
}

// run writes the table examples to an injected stream for deterministic verification.
func run(stdout io.Writer) {
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdout:         stdout,
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

	ascii := false
	console.SetDefault(console.New(console.Config{
		Stdout:         stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &ascii,
	}))
	console.Table(
		[]string{"Status", "Count"},
		[][]string{{"ready", "2"}, {"waiting", "12"}},
		console.TableWidths(8, 5),
		console.TableCenterAlign(0),
		console.TableRightAlign(1),
	)
	// +----------+-------+
	// |  Status  | Count |
	// +----------+-------+
	// |  ready   |     2 |
	// | waiting  |    12 |
	// +----------+-------+
}
