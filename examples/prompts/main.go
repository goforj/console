// Command prompts demonstrates deterministic line and confirmation prompts.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/goforj/console"
)

// main drives prompts with injected input so the complete exchange is reproducible.
func main() {
	run(os.Stdout)
}

// run writes the scripted prompt exchange to an injected stream for exact verification.
func run(stdout io.Writer) {
	var output bytes.Buffer
	interactive := true
	color := false
	unicode := true
	console.SetDefault(console.New(console.Config{
		Stdin:              strings.NewReader("Ada\n\nyes\n"),
		Stdout:             &output,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
	}))

	name, _ := console.Ask("Name")
	environment, _ := console.AskDefault("Environment", "production")
	confirmed, _ := console.Confirm("Deploy now", false)
	fmt.Fprintf(stdout, "%q\n", output.String())
	// "› Name: › Environment [production]: › Deploy now [y/N]: "
	fmt.Fprintln(stdout, name, environment, confirmed)
	// Ada production true
}
