package console_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/goforj/console"
)

// useExampleDefault installs a deterministic package-level console and returns its restore function.
func useExampleDefault(config console.Config) func() {
	previous := console.Default()
	console.SetDefault(console.New(config))
	return func() {
		console.SetDefault(previous)
	}
}

// ExampleAction demonstrates semantic messages and hanging indentation through package-level helpers.
//
// @readme messages
func ExampleAction() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		Stderr:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.Action("Building application")
	// · Building application
	console.Success("API ready\nWorker ready")
	// ✔ API ready
	//   Worker ready
	console.Warn("Configuration is incomplete")
	// ! Configuration is incomplete
	console.Error("Port already in use")
	// ✖ Port already in use

	// Output:
	// · Building application
	// ✔ API ready
	//   Worker ready
	// ! Configuration is incomplete
	// ✖ Port already in use
}

// ExamplePrintln demonstrates ordinary output and writer adapters that cooperate with transient displays.
//
// @readme output
func ExamplePrintln() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		Stderr:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.Println("plain output")
	// plain output
	fmt.Fprintln(console.StdoutWriter(), "streamed output")
	// streamed output
	fmt.Fprintln(console.StderrWriter(), "diagnostic output")
	// diagnostic output

	// Output:
	// plain output
	// streamed output
	// diagnostic output
}

// ExampleStyle demonstrates marks and styles that follow the console's output policy.
//
// @readme styling
func ExampleStyle() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	fmt.Println(console.ActionMark(), console.SuccessMark(), console.ErrorMark())
	// · ✔ ✖
	fmt.Println(console.Style("release ready", console.StyleBold, console.ColorGreen))
	// release ready

	// Output:
	// · ✔ ✖
	// release ready
}

// ExampleSection demonstrates render-only layout helpers for composing deployment summaries.
//
// @readme summaries
func ExampleSection() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
		Width:          24,
	})()
	// @readme:setup:end

	fmt.Println(console.RenderSection("Deployment"))
	// ◇ Deployment
	fmt.Println(console.RenderKeyValues(
		console.KV("Environment", "production"),
		console.KV("Region", "eu-west-1"),
	))
	// Environment  production
	// Region       eu-west-1
	fmt.Println(console.RenderRule("Next"))
	// ── Next ────────────────

	// Output:
	// ◇ Deployment
	// Environment  production
	// Region       eu-west-1
	// ── Next ────────────────
}

// ExampleList demonstrates the two common list presentations.
//
// @readme lists
func ExampleList() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.List("validate configuration", "connect to database")
	// • validate configuration
	// • connect to database
	console.NumberedList("build", "test", "publish")
	// 1. build
	// 2. test
	// 3. publish

	// Output:
	// • validate configuration
	// • connect to database
	// 1. build
	// 2. test
	// 3. publish
}

// ExampleTree demonstrates an ordered static hierarchy with automatic connectors.
//
// @readme trees
func ExampleTree() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.Tree(console.Node("project",
		console.Node("cmd", console.Node("deploy")),
		console.Node("internal"),
		console.Node("README.md"),
	))
	// project
	// ├── cmd
	// │   └── deploy
	// ├── internal
	// └── README.md

	// Output:
	// project
	// ├── cmd
	// │   └── deploy
	// ├── internal
	// └── README.md
}

// ExampleBox demonstrates the useful box defaults and the two common adjustments.
//
// @readme boxes
func ExampleBox() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.Box(
		"The API and worker are healthy.",
		console.BoxTitle("Status"),
		console.BoxWidth(38),
	)
	// ┌─ Status ───────────────────────────┐
	// │ The API and worker are healthy.    │
	// └────────────────────────────────────┘

	// Output:
	// ┌─ Status ───────────────────────────┐
	// │ The API and worker are healthy.    │
	// └────────────────────────────────────┘
}

// ExampleTable demonstrates the bordered table used by default.
//
// @readme tables
func ExampleTable() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

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

	// Output:
	// ┌─────────┬───────┐
	// │ Service │ State │
	// ├─────────┼───────┤
	// │ api     │ ready │
	// │ worker  │ ready │
	// └─────────┴───────┘
}

// ExampleTable_options demonstrates the restrained options for compact, fixed, aligned, and wrapped columns.
//
// @readme table-options
func ExampleTable_options() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

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

	// Output:
	// Task      Seconds
	// ────────  ───────
	// compile        12
	// packages
	// test            3
}

// ExampleTable_ascii demonstrates the ASCII fallback and centered columns for constrained terminals.
//
// @readme table-ascii
func ExampleTable_ascii() {
	// @readme:setup:start
	color := false
	unicode := false
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

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

	// Output:
	// +----------+-------+
	// |  Status  | Count |
	// +----------+-------+
	// |  ready   |     2 |
	// | waiting  |    12 |
	// +----------+-------+
}

// ExampleNewLoader demonstrates stable loader outcomes when output is redirected.
//
// @readme loader
func ExampleNewLoader() {
	// @readme:setup:start
	color := false
	animations := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:            os.Stdout,
		Stderr:            os.Stdout,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	})()
	// @readme:setup:end

	download := console.NewLoader("Downloading modules")
	if err := download.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Downloading modules
	defer download.Stop()
	download.Success("Modules ready")
	// ✔ Modules ready

	publish := console.NewLoader("Publishing release")
	if err := publish.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Publishing release
	defer publish.Stop()
	publish.Fail("Registry refused upload")
	// ✖ Registry refused upload

	// Output:
	// · Downloading modules
	// ✔ Modules ready
	// · Publishing release
	// ✖ Registry refused upload
}

// ExampleNewProgress demonstrates determinate work with a durable redirected-output contract.
//
// @readme progress
func ExampleNewProgress() {
	// @readme:setup:start
	color := false
	animations := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:            os.Stdout,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	})()
	// @readme:setup:end

	progress := console.NewProgress(100, "Packaging release")
	if err := progress.Start(); err != nil {
		console.Error(err.Error())
		return
	}
	// · Packaging release
	defer progress.Stop()
	progress.Step(40, "Uploading release")
	progress.Add(60)
	progress.Complete("Release ready")
	// ✔ Release ready

	// Output:
	// · Packaging release
	// ✔ Release ready
}

// ExampleAsk demonstrates the common line, default, and confirmation prompts with scripted input.
//
// @readme prompts
func ExampleAsk() {
	var output bytes.Buffer
	interactive := true
	color := false
	unicode := true
	// @readme:setup:start
	previous := console.Default()
	defer console.SetDefault(previous)
	// @readme:setup:end
	console.SetDefault(console.New(console.Config{
		Stdin:              strings.NewReader("Ada\n\nyes\n"),
		Stdout:             &output,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
	}))

	name, _ := console.Ask("Name")
	fmt.Println(strings.TrimSpace(output.String()))
	// › Name:
	output.Reset()
	environment, _ := console.AskDefault("Environment", "production")
	fmt.Println(strings.TrimSpace(output.String()))
	// › Environment [production]:
	output.Reset()
	confirmed, _ := console.Confirm("Deploy now", false)
	fmt.Println(strings.TrimSpace(output.String()))
	// › Deploy now [y/N]:
	fmt.Println(name, environment, confirmed)
	// Ada production true

	// Output:
	// › Name:
	// › Environment [production]:
	// › Deploy now [y/N]:
	// Ada production true
}

// ExampleChoose demonstrates a numbered choice and non-echoed secret input.
//
// @readme selection
func ExampleChoose() {
	var output bytes.Buffer
	interactive := true
	color := false
	unicode := true
	// @readme:setup:start
	previous := console.Default()
	defer console.SetDefault(previous)
	// @readme:setup:end
	console.SetDefault(console.New(console.Config{
		Stdin:              strings.NewReader("2"),
		Stdout:             &output,
		InteractiveEnabled: &interactive,
		ColorEnabled:       &color,
		UnicodeEnabled:     &unicode,
		ReadSecret: func() (string, error) {
			return "token-value", nil
		},
	}))

	channel, _ := console.Choose("Release channel", []string{"stable", "beta"}, 0)
	fmt.Println(strings.TrimSpace(output.String()))
	// Release channel
	// 1. stable
	// 2. beta
	// › Choose [1-2, default 1]:
	output.Reset()
	secret, _ := console.AskSecret("API token")
	fmt.Println(strings.TrimSpace(output.String()))
	// › API token:
	fmt.Println(channel, len(secret))
	// beta 11

	// Output:
	// Release channel
	// 1. stable
	// 2. beta
	// › Choose [1-2, default 1]:
	// › API token:
	// beta 11
}

// ExampleStripANSI demonstrates terminal-cell-aware shaping for plain and styled text.
//
// @readme text
func ExampleStripANSI() {
	styled := "\x1b[31mGo 世界\x1b[0m"

	fmt.Println(console.StripANSI(styled))
	// Go 世界
	fmt.Println(console.VisibleWidth(styled))
	// 7
	fmt.Printf("%q\n", console.PadLeft("Go", 6))
	// "    Go"
	fmt.Printf("%q\n", console.PadCenter("Go", 6))
	// "  Go  "
	fmt.Println(console.TruncateMiddle("github.com/goforj/console", 15))
	// github.…console
	fmt.Println(console.Wrap("deploying worker service", 10))
	// deploying
	// worker
	// service

	// Output:
	// Go 世界
	// 7
	// "    Go"
	// "  Go  "
	// github.…console
	// deploying
	// worker
	// service
}

// Example_deploymentRecipe demonstrates a complete deployment lifecycle with concise package helpers.
//
// @readme deployment-recipe
func Example_deploymentRecipe() {
	// @readme:setup:start
	color := false
	animations := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:            os.Stdout,
		Stderr:            os.Stdout,
		ColorEnabled:      &color,
		UnicodeEnabled:    &unicode,
		AnimationsEnabled: &animations,
	})()
	// @readme:setup:end

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

	// Output:
	// ◇ Deploy production
	// Environment  production
	// Region       eu-west-1
	// · Deploying services
	// ✔ Services deployed
	// ┌─────────┬───────┐
	// │ Service │ State │
	// ├─────────┼───────┤
	// │ api     │ ready │
	// │ worker  │ ready │
	// └─────────┴───────┘
	// ✔ Deployment complete
}

// Example_validationRecipe demonstrates a compact report that tells users what to fix next.
//
// @readme validation-recipe
func Example_validationRecipe() {
	// @readme:setup:start
	color := false
	unicode := true
	defer useExampleDefault(console.Config{
		Stdout:         os.Stdout,
		Stderr:         os.Stdout,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})()
	// @readme:setup:end

	console.Section("Configuration check")
	// ◇ Configuration check
	console.KeyValues(
		console.KV("Checks", 8),
		console.KV("Passed", 6),
		console.KV("Failed", 2),
	)
	// Checks  8
	// Passed  6
	// Failed  2
	console.Warn("2 issues need attention")
	// ! 2 issues need attention
	console.List("DATABASE_URL is missing", "PORT must be between 1 and 65535")
	// • DATABASE_URL is missing
	// • PORT must be between 1 and 65535
	console.Error("Validation failed")
	// ✖ Validation failed

	// Output:
	// ◇ Configuration check
	// Checks  8
	// Passed  6
	// Failed  2
	// ! 2 issues need attention
	// • DATABASE_URL is missing
	// • PORT must be between 1 and 65535
	// ✖ Validation failed
}

// Example_ciRecipe demonstrates keeping machine output separate from human-facing CI status.
//
// @readme ci-recipe
func Example_ciRecipe() {
	var machineOutput bytes.Buffer
	var statusOutput bytes.Buffer
	color := false
	unicode := false
	// @readme:setup:start
	previous := console.Default()
	defer console.SetDefault(previous)
	// @readme:setup:end
	console.SetDefault(console.New(console.Config{
		Stdout:         &machineOutput,
		Stderr:         &statusOutput,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	}))

	fmt.Fprintln(console.StdoutWriter(), `{"artifact":"app.tar.gz","status":"ready"}`)
	fmt.Fprintln(console.StderrWriter(), "status: uploading app.tar.gz")
	fmt.Println("stdout:")
	// stdout:
	fmt.Print(machineOutput.String())
	// {"artifact":"app.tar.gz","status":"ready"}
	fmt.Println("stderr:")
	// stderr:
	fmt.Print(statusOutput.String())
	// status: uploading app.tar.gz

	// Output:
	// stdout:
	// {"artifact":"app.tar.gz","status":"ready"}
	// stderr:
	// status: uploading app.tar.gz
}

// ExampleNew demonstrates an isolated console for libraries and independently configured commands.
//
// @readme instance
func ExampleNew() {
	var output bytes.Buffer
	color := false
	unicode := false
	commandConsole := console.New(console.Config{
		Stdout:         &output,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
	})

	commandConsole.Success("Isolated output")
	fmt.Print(output.String())
	// + Isolated output

	// Output:
	// + Isolated output
}
