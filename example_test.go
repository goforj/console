package console_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goforj/console"
)

// ExampleConsole demonstrates deterministic semantic output with isolated writers.
//
// @readme messages
func ExampleConsole() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	color := false
	unicode := true
	cli := console.New(console.Config{
		Stdout:         &stdout,
		Stderr:         &stderr,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})

	cli.Action("Building application")
	cli.Success("Application ready")
	cli.Error("Port already in use")

	fmt.Print(stdout.String())
	fmt.Print(stderr.String())
	// Output:
	// · Building application
	// ✔ Application ready
	// ✖ Port already in use
}

// ExampleConsole_layout demonstrates composable layout with deterministic terminal policy.
//
// @readme layout
func ExampleConsole_layout() {
	var output bytes.Buffer
	color := false
	unicode := true
	cli := console.New(console.Config{
		Stdout:         &output,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
		Width:          32,
	})

	cli.List("api ready", "worker ready")
	cli.Box("All services healthy.", console.BoxTitle("Status"), console.BoxWidth(26))
	cli.Table(
		[]string{"Service", "State"},
		[][]string{{"api", "ready"}, {"worker", "ready"}},
	)

	fmt.Print(output.String())
	// Output:
	// • api ready
	// • worker ready
	// ┌─ Status ───────────────┐
	// │ All services healthy.  │
	// └────────────────────────┘
	// ┌─────────┬───────┐
	// │ Service │ State │
	// ├─────────┼───────┤
	// │ api     │ ready │
	// │ worker  │ ready │
	// └─────────┴───────┘
}

// ExampleLoader demonstrates the redirected loader contract without timing or terminal state.
//
// @readme loader
func ExampleLoader() {
	var output bytes.Buffer
	color := false
	animations := false
	unicode := true
	cli := console.New(console.Config{
		Stdout:            &output,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	})

	loader := cli.Loader("Downloading modules")
	_ = loader.Start()
	loader.Success("Modules ready")

	fmt.Print(output.String())
	// Output:
	// · Downloading modules
	// ✔ Modules ready
}

// ExampleConsole_Confirm demonstrates scripted prompt input with an explicit interactive override.
//
// @readme prompts
func ExampleConsole_Confirm() {
	var output bytes.Buffer
	interactive := true
	color := false
	unicode := true
	cli := console.New(console.Config{
		Stdin:              strings.NewReader("yes\n"),
		Stdout:             &output,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
	})

	confirmed, err := cli.Confirm("Deploy now", false)
	fmt.Printf("%q\n", output.String())
	fmt.Println(confirmed, err)
	// Output:
	// "› Deploy now [y/N]: "
	// true <nil>
}

// ExampleStripANSI demonstrates ANSI-aware measurement and text shaping.
//
// @readme text
func ExampleStripANSI() {
	styled := "\x1b[31mGo 世界\x1b[0m"

	fmt.Println(console.StripANSI(styled))
	fmt.Println(console.VisibleWidth(styled))
	fmt.Println(console.Truncate("deploying worker", 10))
	fmt.Println(console.Wrap("deploying worker service", 10))
	// Output:
	// Go 世界
	// 7
	// deploying…
	// deploying
	// worker
	// service
}
