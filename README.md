<p align="center">
  <strong>console</strong>
</p>

<p align="center">
  Lightweight building blocks for polished Go console output.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/goforj/console"><img src="https://pkg.go.dev/badge/github.com/goforj/console.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://github.com/goforj/console/actions"><img src="https://github.com/goforj/console/actions/workflows/test.yml/badge.svg" alt="Go Test"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-1.24%2B-blue?logo=go" alt="Go 1.24 or newer"></a>
  <img src="https://img.shields.io/github/v/tag/goforj/console?label=version&sort=semver" alt="Latest tag">
  <a href="https://codecov.io/gh/goforj/console"><img src="https://codecov.io/github/goforj/console/graph/badge.svg" alt="Coverage"></a>
  <a href="https://goreportcard.com/report/github.com/goforj/console"><img src="https://goreportcard.com/badge/github.com/goforj/console" alt="Go Report Card"></a>
</p>

`console` is a small toolkit for the output layer shared by command-line applications: semantic messages, ANSI styling, terminal-cell-aware text, boxes, tables, lists, prompts, and one-line loaders. It is deliberately not a full-screen TUI framework. There is no event loop, raw-mode ownership, command parser, or logging pipeline to adopt.

## Installation

Requires Go 1.24 or newer.

```sh
go get github.com/goforj/console
```

## Quick start

```go
package main

import "github.com/goforj/console"

func main() {
	console.Action("Building application")
	console.Infof("Environment: %s", "development")
	console.Success("Application ready")
}

// · Building application
// · Environment: development
// ✔ Application ready
```

Common operations are available both as package-level helpers and as methods on `*Console`. Package helpers use the process-wide default console; construct an instance when a command needs isolated writers or deterministic behavior. Loader construction is the intentional naming exception: use `console.NewLoader("message")` with the default console or `cli.Loader("message")` with an instance.

```go
var output bytes.Buffer
color := false

cli := console.New(console.Config{
	Stdout:       &output,
	Stderr:       &output,
	ColorEnabled: &color,
})

cli.Warn("Configuration is incomplete")
fmt.Print(output.String())

// ! Configuration is incomplete
```

Set `ColorEnabled` explicitly when output policy should ignore environment and TTY detection:

```go
var output bytes.Buffer
forceColor := true
unicode := true
colored := console.New(console.Config{
	Stdout:         &output,
	ColorEnabled:   &forceColor,
	UnicodeEnabled: &unicode,
})
colored.Success("ANSI styling is forced")
fmt.Printf("%q\n", output.String())

// "\x1b[32m✔\x1b[0m ANSI styling is forced\n"
```

## What it provides

- Semantic action, information, success, warning, error, fatal, and debug messages.
- Automatic ANSI color policy with `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, and TTY awareness.
- ANSI-aware visible width, truncation, wrapping, indentation, and padding helpers.
- Sections, rules, ordered key/value rows, lists, boxes, and ragged-safe tables.
- Line-oriented `Ask`, `Confirm`, and numbered `Choose` prompts.
- A concurrency-safe, single-line loader that turns into stable log lines when output is redirected.
- Configurable writers, input, marks, terminal hooks, environment lookup, and exit behavior.

## Design principles

- **Stay lightweight.** The only direct runtime dependency is `golang.org/x/term`.
- **Keep output composable.** `Console` owns presentation policy, not structured logging, command routing, or application lifecycle.
- **Prefer durable logs outside a TTY.** Loaders never leak carriage returns or erase sequences into redirected output.
- **Treat layout as terminal cells.** ANSI sequences are zero-width; combining marks, CJK text, and common emoji are measured for console alignment.
- **Make testing ordinary.** Writers, input, environment lookup, terminal detection, terminal size, and process exit are injectable.
- **Fail fast on invalid wiring.** `SetDefault(nil)` panics instead of leaving package helpers silently unusable.

## Layout

Layout helpers write through a `Console`; their `RenderBox` and `RenderTable` counterparts return strings for composition.

```go
color := false
unicode := true
cli := console.New(console.Config{
	ColorEnabled:   &color,
	UnicodeEnabled: &unicode,
})

fmt.Println(cli.RenderBox(
	"All services healthy.",
	console.BoxTitle("Status"),
	console.BoxWidth(26),
))

// ┌─ Status ───────────────┐
// │ All services healthy.  │
// └────────────────────────┘
```

Borders, marks, list bullets, and loader frames have ASCII fallbacks. Set `UnicodeEnabled` to `false` when targeting a constrained terminal; text measurement remains Unicode-aware.

## Loaders

Constructing a loader has no side effects. `Start` claims the console's transient line when animation is possible; a second animated loader receives `ErrLoaderActive`.

```go
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
if err := loader.Start(); err != nil {
	cli.Error(err.Error())
	return
}

loader.Update("Verifying modules")
loader.Success("Modules ready")
fmt.Print(output.String())

// · Downloading modules
// ✔ Modules ready
```

On a terminal this animates in place. Redirected output uses the durable semantic lines shown in the example.

`Stop`, `Success`, `Warn`, and `Fail` are idempotent terminal operations; the first one wins. Complete output lines temporarily clear and redraw an active loader. Partial `Print`/`Printf` output and prompts pause animation until the line completes or input returns.

## Prompts

Prompts refuse to read when the configured input and output are not terminals, returning `ErrNonInteractive` instead of unexpectedly blocking automation. Tests and intentional scripted input can opt in explicitly:

```go
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

// "› Deploy now [y/N]: "
// true <nil>
```

The prompt reader is retained for the lifetime of the console, so sequential prompts do not lose input to buffering.
While a prompt waits for input, complete writes from other goroutines wait rather than overwrite the live input line.

## Output behavior

Action, info, success, warning, and debug messages go to stdout. Error and fatal messages go to stderr. `Fatal` and `Fatalf` are the only helpers that exit; their exit function is configurable.

Color selection follows this precedence:

1. `Config.ColorEnabled` when set.
2. `NO_COLOR` disables styling.
3. A nonzero `CLICOLOR_FORCE` enables styling.
4. `CLICOLOR=0` or `TERM=dumb` disables styling.
5. Otherwise stdout must expose a terminal file descriptor.

Debug output is enabled by a nonzero `FORJ_DEBUG`, `APP_DEBUG`, or `DEBUG` value unless `Config.DebugEnabled` overrides it.

## Runnable examples

| Example | Command |
| --- | --- |
| Semantic messages and custom writers | `go -C examples run ./messages` |
| Boxes, key/value rows, and tables | `go -C examples run ./layout` |
| Redirect-safe loader lifecycle | `go -C examples run ./loader` |

<!-- api:embed:start -->

## API index

The complete API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/goforj/console).

| Group | API |
| --- | --- |
| Boxes | <a id="box"></a>[Box](https://pkg.go.dev/github.com/goforj/console#Box) · <a id="boxcolor"></a>[BoxColor](https://pkg.go.dev/github.com/goforj/console#BoxColor) · <a id="boxoption"></a>[BoxOption](https://pkg.go.dev/github.com/goforj/console#BoxOption) · <a id="boxpadding"></a>[BoxPadding](https://pkg.go.dev/github.com/goforj/console#BoxPadding) · <a id="boxtitle"></a>[BoxTitle](https://pkg.go.dev/github.com/goforj/console#BoxTitle) · <a id="boxwidth"></a>[BoxWidth](https://pkg.go.dev/github.com/goforj/console#BoxWidth) · <a id="console-box"></a>[Console.Box](https://pkg.go.dev/github.com/goforj/console#Console.Box) · <a id="console-renderbox"></a>[Console.RenderBox](https://pkg.go.dev/github.com/goforj/console#Console.RenderBox) · <a id="renderbox"></a>[RenderBox](https://pkg.go.dev/github.com/goforj/console#RenderBox) |
| Layout | <a id="console-keyvaluemap"></a>[Console.KeyValueMap](https://pkg.go.dev/github.com/goforj/console#Console.KeyValueMap) · <a id="console-keyvalues"></a>[Console.KeyValues](https://pkg.go.dev/github.com/goforj/console#Console.KeyValues) · <a id="console-list"></a>[Console.List](https://pkg.go.dev/github.com/goforj/console#Console.List) · <a id="console-numberedlist"></a>[Console.NumberedList](https://pkg.go.dev/github.com/goforj/console#Console.NumberedList) · <a id="console-rule"></a>[Console.Rule](https://pkg.go.dev/github.com/goforj/console#Console.Rule) · <a id="console-section"></a>[Console.Section](https://pkg.go.dev/github.com/goforj/console#Console.Section) · <a id="kv"></a>[KV](https://pkg.go.dev/github.com/goforj/console#KV) · <a id="keyvalue"></a>[KeyValue](https://pkg.go.dev/github.com/goforj/console#KeyValue) · <a id="keyvaluemap"></a>[KeyValueMap](https://pkg.go.dev/github.com/goforj/console#KeyValueMap) · <a id="keyvalues"></a>[KeyValues](https://pkg.go.dev/github.com/goforj/console#KeyValues) · <a id="list"></a>[List](https://pkg.go.dev/github.com/goforj/console#List) · <a id="numberedlist"></a>[NumberedList](https://pkg.go.dev/github.com/goforj/console#NumberedList) · <a id="rule"></a>[Rule](https://pkg.go.dev/github.com/goforj/console#Rule) · <a id="section"></a>[Section](https://pkg.go.dev/github.com/goforj/console#Section) |
| Loaders | <a id="console-loader"></a>[Console.Loader](https://pkg.go.dev/github.com/goforj/console#Console.Loader) · <a id="errloaderactive"></a>[ErrLoaderActive](https://pkg.go.dev/github.com/goforj/console#ErrLoaderActive) · <a id="loader"></a>[Loader](https://pkg.go.dev/github.com/goforj/console#Loader) · <a id="loader-fail"></a>[Loader.Fail](https://pkg.go.dev/github.com/goforj/console#Loader.Fail) · <a id="loader-start"></a>[Loader.Start](https://pkg.go.dev/github.com/goforj/console#Loader.Start) · <a id="loader-stop"></a>[Loader.Stop](https://pkg.go.dev/github.com/goforj/console#Loader.Stop) · <a id="loader-success"></a>[Loader.Success](https://pkg.go.dev/github.com/goforj/console#Loader.Success) · <a id="loader-update"></a>[Loader.Update](https://pkg.go.dev/github.com/goforj/console#Loader.Update) · <a id="loader-warn"></a>[Loader.Warn](https://pkg.go.dev/github.com/goforj/console#Loader.Warn) · <a id="newloader"></a>[NewLoader](https://pkg.go.dev/github.com/goforj/console#NewLoader) |
| Marks | <a id="actionmark"></a>[ActionMark](https://pkg.go.dev/github.com/goforj/console#ActionMark) · <a id="console-actionmark"></a>[Console.ActionMark](https://pkg.go.dev/github.com/goforj/console#Console.ActionMark) · <a id="console-debugmark"></a>[Console.DebugMark](https://pkg.go.dev/github.com/goforj/console#Console.DebugMark) · <a id="console-errormark"></a>[Console.ErrorMark](https://pkg.go.dev/github.com/goforj/console#Console.ErrorMark) · <a id="console-infomark"></a>[Console.InfoMark](https://pkg.go.dev/github.com/goforj/console#Console.InfoMark) · <a id="console-successmark"></a>[Console.SuccessMark](https://pkg.go.dev/github.com/goforj/console#Console.SuccessMark) · <a id="console-warnmark"></a>[Console.WarnMark](https://pkg.go.dev/github.com/goforj/console#Console.WarnMark) · <a id="debugmark"></a>[DebugMark](https://pkg.go.dev/github.com/goforj/console#DebugMark) · <a id="errormark"></a>[ErrorMark](https://pkg.go.dev/github.com/goforj/console#ErrorMark) · <a id="infomark"></a>[InfoMark](https://pkg.go.dev/github.com/goforj/console#InfoMark) · <a id="successmark"></a>[SuccessMark](https://pkg.go.dev/github.com/goforj/console#SuccessMark) · <a id="warnmark"></a>[WarnMark](https://pkg.go.dev/github.com/goforj/console#WarnMark) |
| Messages | <a id="action"></a>[Action](https://pkg.go.dev/github.com/goforj/console#Action) · <a id="actionf"></a>[Actionf](https://pkg.go.dev/github.com/goforj/console#Actionf) · <a id="console-action"></a>[Console.Action](https://pkg.go.dev/github.com/goforj/console#Console.Action) · <a id="console-actionf"></a>[Console.Actionf](https://pkg.go.dev/github.com/goforj/console#Console.Actionf) · <a id="console-debug"></a>[Console.Debug](https://pkg.go.dev/github.com/goforj/console#Console.Debug) · <a id="console-debugf"></a>[Console.Debugf](https://pkg.go.dev/github.com/goforj/console#Console.Debugf) · <a id="console-error"></a>[Console.Error](https://pkg.go.dev/github.com/goforj/console#Console.Error) · <a id="console-errorf"></a>[Console.Errorf](https://pkg.go.dev/github.com/goforj/console#Console.Errorf) · <a id="console-fatal"></a>[Console.Fatal](https://pkg.go.dev/github.com/goforj/console#Console.Fatal) · <a id="console-fatalf"></a>[Console.Fatalf](https://pkg.go.dev/github.com/goforj/console#Console.Fatalf) · <a id="console-info"></a>[Console.Info](https://pkg.go.dev/github.com/goforj/console#Console.Info) · <a id="console-infof"></a>[Console.Infof](https://pkg.go.dev/github.com/goforj/console#Console.Infof) · <a id="console-success"></a>[Console.Success](https://pkg.go.dev/github.com/goforj/console#Console.Success) · <a id="console-successf"></a>[Console.Successf](https://pkg.go.dev/github.com/goforj/console#Console.Successf) · <a id="console-warn"></a>[Console.Warn](https://pkg.go.dev/github.com/goforj/console#Console.Warn) · <a id="console-warnf"></a>[Console.Warnf](https://pkg.go.dev/github.com/goforj/console#Console.Warnf) · <a id="debug"></a>[Debug](https://pkg.go.dev/github.com/goforj/console#Debug) · <a id="debugf"></a>[Debugf](https://pkg.go.dev/github.com/goforj/console#Debugf) · <a id="error"></a>[Error](https://pkg.go.dev/github.com/goforj/console#Error) · <a id="errorf"></a>[Errorf](https://pkg.go.dev/github.com/goforj/console#Errorf) · <a id="fatal"></a>[Fatal](https://pkg.go.dev/github.com/goforj/console#Fatal) · <a id="fatalf"></a>[Fatalf](https://pkg.go.dev/github.com/goforj/console#Fatalf) · <a id="info"></a>[Info](https://pkg.go.dev/github.com/goforj/console#Info) · <a id="infof"></a>[Infof](https://pkg.go.dev/github.com/goforj/console#Infof) · <a id="success"></a>[Success](https://pkg.go.dev/github.com/goforj/console#Success) · <a id="successf"></a>[Successf](https://pkg.go.dev/github.com/goforj/console#Successf) · <a id="warn"></a>[Warn](https://pkg.go.dev/github.com/goforj/console#Warn) · <a id="warnf"></a>[Warnf](https://pkg.go.dev/github.com/goforj/console#Warnf) |
| Output | <a id="console-newline"></a>[Console.NewLine](https://pkg.go.dev/github.com/goforj/console#Console.NewLine) · <a id="console-print"></a>[Console.Print](https://pkg.go.dev/github.com/goforj/console#Console.Print) · <a id="console-printf"></a>[Console.Printf](https://pkg.go.dev/github.com/goforj/console#Console.Printf) · <a id="console-println"></a>[Console.Println](https://pkg.go.dev/github.com/goforj/console#Console.Println) · <a id="newline"></a>[NewLine](https://pkg.go.dev/github.com/goforj/console#NewLine) · <a id="print"></a>[Print](https://pkg.go.dev/github.com/goforj/console#Print) · <a id="printf"></a>[Printf](https://pkg.go.dev/github.com/goforj/console#Printf) · <a id="println"></a>[Println](https://pkg.go.dev/github.com/goforj/console#Println) |
| Prompts | <a id="ask"></a>[Ask](https://pkg.go.dev/github.com/goforj/console#Ask) · <a id="askdefault"></a>[AskDefault](https://pkg.go.dev/github.com/goforj/console#AskDefault) · <a id="choose"></a>[Choose](https://pkg.go.dev/github.com/goforj/console#Choose) · <a id="confirm"></a>[Confirm](https://pkg.go.dev/github.com/goforj/console#Confirm) · <a id="console-ask"></a>[Console.Ask](https://pkg.go.dev/github.com/goforj/console#Console.Ask) · <a id="console-askdefault"></a>[Console.AskDefault](https://pkg.go.dev/github.com/goforj/console#Console.AskDefault) · <a id="console-choose"></a>[Console.Choose](https://pkg.go.dev/github.com/goforj/console#Console.Choose) · <a id="console-confirm"></a>[Console.Confirm](https://pkg.go.dev/github.com/goforj/console#Console.Confirm) · <a id="errnoninteractive"></a>[ErrNonInteractive](https://pkg.go.dev/github.com/goforj/console#ErrNonInteractive) |
| Runtime | <a id="asciimarks"></a>[ASCIIMarks](https://pkg.go.dev/github.com/goforj/console#ASCIIMarks) · <a id="config"></a>[Config](https://pkg.go.dev/github.com/goforj/console#Config) · <a id="console"></a>[Console](https://pkg.go.dev/github.com/goforj/console#Console) · <a id="default"></a>[Default](https://pkg.go.dev/github.com/goforj/console#Default) · <a id="defaultmarks"></a>[DefaultMarks](https://pkg.go.dev/github.com/goforj/console#DefaultMarks) · <a id="marks"></a>[Marks](https://pkg.go.dev/github.com/goforj/console#Marks) · <a id="new"></a>[New](https://pkg.go.dev/github.com/goforj/console#New) · <a id="setdefault"></a>[SetDefault](https://pkg.go.dev/github.com/goforj/console#SetDefault) |
| Styling | <a id="colorblack"></a>[ColorBlack](https://pkg.go.dev/github.com/goforj/console#ColorBlack) · <a id="colorblue"></a>[ColorBlue](https://pkg.go.dev/github.com/goforj/console#ColorBlue) · <a id="colorboldgreen"></a>[ColorBoldGreen](https://pkg.go.dev/github.com/goforj/console#ColorBoldGreen) · <a id="colorboldwhite"></a>[ColorBoldWhite](https://pkg.go.dev/github.com/goforj/console#ColorBoldWhite) · <a id="colorcyan"></a>[ColorCyan](https://pkg.go.dev/github.com/goforj/console#ColorCyan) · <a id="colorgray"></a>[ColorGray](https://pkg.go.dev/github.com/goforj/console#ColorGray) · <a id="colorgreen"></a>[ColorGreen](https://pkg.go.dev/github.com/goforj/console#ColorGreen) · <a id="colormagenta"></a>[ColorMagenta](https://pkg.go.dev/github.com/goforj/console#ColorMagenta) · <a id="colorred"></a>[ColorRed](https://pkg.go.dev/github.com/goforj/console#ColorRed) · <a id="colorreset"></a>[ColorReset](https://pkg.go.dev/github.com/goforj/console#ColorReset) · <a id="colorwhite"></a>[ColorWhite](https://pkg.go.dev/github.com/goforj/console#ColorWhite) · <a id="coloryellow"></a>[ColorYellow](https://pkg.go.dev/github.com/goforj/console#ColorYellow) · <a id="colorize"></a>[Colorize](https://pkg.go.dev/github.com/goforj/console#Colorize) · <a id="console-colorize"></a>[Console.Colorize](https://pkg.go.dev/github.com/goforj/console#Console.Colorize) · <a id="console-style"></a>[Console.Style](https://pkg.go.dev/github.com/goforj/console#Console.Style) · <a id="style"></a>[Style](https://pkg.go.dev/github.com/goforj/console#Style) · <a id="stylebold"></a>[StyleBold](https://pkg.go.dev/github.com/goforj/console#StyleBold) · <a id="styledim"></a>[StyleDim](https://pkg.go.dev/github.com/goforj/console#StyleDim) · <a id="styleunderline"></a>[StyleUnderline](https://pkg.go.dev/github.com/goforj/console#StyleUnderline) |
| Tables | <a id="console-rendertable"></a>[Console.RenderTable](https://pkg.go.dev/github.com/goforj/console#Console.RenderTable) · <a id="console-table"></a>[Console.Table](https://pkg.go.dev/github.com/goforj/console#Console.Table) · <a id="rendertable"></a>[RenderTable](https://pkg.go.dev/github.com/goforj/console#RenderTable) · <a id="table"></a>[Table](https://pkg.go.dev/github.com/goforj/console#Table) |
| Terminal | <a id="console-isinteractive"></a>[Console.IsInteractive](https://pkg.go.dev/github.com/goforj/console#Console.IsInteractive) · <a id="console-supportscolor"></a>[Console.SupportsColor](https://pkg.go.dev/github.com/goforj/console#Console.SupportsColor) · <a id="console-supportsunicode"></a>[Console.SupportsUnicode](https://pkg.go.dev/github.com/goforj/console#Console.SupportsUnicode) · <a id="console-width"></a>[Console.Width](https://pkg.go.dev/github.com/goforj/console#Console.Width) · <a id="isinteractive"></a>[IsInteractive](https://pkg.go.dev/github.com/goforj/console#IsInteractive) · <a id="supportscolor"></a>[SupportsColor](https://pkg.go.dev/github.com/goforj/console#SupportsColor) · <a id="supportsunicode"></a>[SupportsUnicode](https://pkg.go.dev/github.com/goforj/console#SupportsUnicode) · <a id="width"></a>[Width](https://pkg.go.dev/github.com/goforj/console#Width) |
| Text | <a id="console-expandtabs"></a>[Console.ExpandTabs](https://pkg.go.dev/github.com/goforj/console#Console.ExpandTabs) · <a id="console-indent"></a>[Console.Indent](https://pkg.go.dev/github.com/goforj/console#Console.Indent) · <a id="console-padright"></a>[Console.PadRight](https://pkg.go.dev/github.com/goforj/console#Console.PadRight) · <a id="console-stripansi"></a>[Console.StripANSI](https://pkg.go.dev/github.com/goforj/console#Console.StripANSI) · <a id="console-truncate"></a>[Console.Truncate](https://pkg.go.dev/github.com/goforj/console#Console.Truncate) · <a id="console-visiblewidth"></a>[Console.VisibleWidth](https://pkg.go.dev/github.com/goforj/console#Console.VisibleWidth) · <a id="console-wrap"></a>[Console.Wrap](https://pkg.go.dev/github.com/goforj/console#Console.Wrap) · <a id="expandtabs"></a>[ExpandTabs](https://pkg.go.dev/github.com/goforj/console#ExpandTabs) · <a id="indent"></a>[Indent](https://pkg.go.dev/github.com/goforj/console#Indent) · <a id="padright"></a>[PadRight](https://pkg.go.dev/github.com/goforj/console#PadRight) · <a id="stripansi"></a>[StripANSI](https://pkg.go.dev/github.com/goforj/console#StripANSI) · <a id="truncate"></a>[Truncate](https://pkg.go.dev/github.com/goforj/console#Truncate) · <a id="visiblewidth"></a>[VisibleWidth](https://pkg.go.dev/github.com/goforj/console#VisibleWidth) · <a id="wrap"></a>[Wrap](https://pkg.go.dev/github.com/goforj/console#Wrap) |

## Executable examples

These programs are generated from standard Go example tests. The test suite executes each one and verifies the output appended as comments.

### Semantic messages and custom writers

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/goforj/console"
)

func main() {
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
}

// · Building application
// ✔ Application ready
// ✖ Port already in use
```

### Boxes, tables, and lists

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/goforj/console"
)

func main() {
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
}

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
```

### Redirect-safe loader lifecycle

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/goforj/console"
)

func main() {
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
}

// · Downloading modules
// ✔ Modules ready
```

### Scripted prompts

```go
package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goforj/console"
)

func main() {
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
}

// "› Deploy now [y/N]: "
// true <nil>
```

### ANSI-aware text utilities

```go
package main

import (
	"fmt"

	"github.com/goforj/console"
)

func main() {
	styled := "\x1b[31mGo 世界\x1b[0m"

	fmt.Println(console.StripANSI(styled))
	fmt.Println(console.VisibleWidth(styled))
	fmt.Println(console.Truncate("deploying worker", 10))
	fmt.Println(console.Wrap("deploying worker service", 10))
}

// Go 世界
// 7
// deploying…
// deploying
// worker
// service
```
<!-- api:embed:end -->

## Development

```sh
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go test ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go test -race ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go -C docs test ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go -C examples test ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go generate .
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go vet ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go -C docs vet ./...
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go -C examples vet ./...
```

The docs and examples are separate Go modules so release archives contain only the library. The README API index is generated from public GoDoc and `@group` metadata; its representative programs and output come from standard Go example tests. Generation validates its marker pair before writing, so malformed hand-edited structure fails without partially changing the document.

## License

Released under the [MIT License](LICENSE).
