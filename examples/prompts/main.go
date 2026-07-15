// Command prompts demonstrates deterministic line and confirmation prompts.
package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goforj/console"
)

// main drives prompts with injected input so the complete exchange is reproducible.
func main() {
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
	fmt.Printf("%q\n", output.String())
	// "› Name: › Environment [production]: › Deploy now [y/N]: "
	fmt.Println(name, environment, confirmed)
	// Ada production true
}
