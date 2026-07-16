// Command ci demonstrates clean machine stdout with human status on stderr.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/goforj/console"
)

// main writes one machine record and one human-facing status line.
func main() {
	run(os.Stdout, os.Stderr)
}

// run keeps the two output contracts independently testable.
func run(stdout, stderr io.Writer) {
	color := false
	unicode := false
	console.SetDefault(console.New(console.Config{
		Stdout:         stdout,
		Stderr:         stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

	fmt.Fprintln(console.StdoutWriter(), `{"artifact":"app.tar.gz","status":"ready"}`)
	// stdout: {"artifact":"app.tar.gz","status":"ready"}
	fmt.Fprintln(console.StderrWriter(), "status: uploading app.tar.gz")
	// stderr: status: uploading app.tar.gz
}
