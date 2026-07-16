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

`console` is a small toolkit for the output layer shared by command-line applications: semantic messages, coordinated writers, ANSI-aware text, composable layout, bordered and compact tables, trees, prompts, loaders, and progress. It is deliberately not a full-screen TUI framework. There is no event loop, raw-mode ownership, command parser, subprocess runner, or logging pipeline to adopt.

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
	// · Building application
	console.Infof("Environment: %s", "development")
	// · Environment: development
	console.Success("Application ready")
	// ✔ Application ready
}
```

Common operations are available both as package-level helpers and as methods on `*Console`. The examples lead with the concise package helpers, which use the process-wide default console. Libraries and applications that need isolated writers or independent runtime policy should construct an instance instead. Construction is the only naming exception: package helpers use `console.NewLoader` and `console.NewProgress`, while instances use `Loader` and `Progress`.

```go
var output bytes.Buffer
color := false

console.SetDefault(console.New(console.Config{
	Stdout:       &output,
	Stderr:       &output,
	ColorEnabled: &color,
}))

console.Warn("Configuration is incomplete")
fmt.Print(output.String())
// ! Configuration is incomplete
```

Set `ColorEnabled` explicitly when output policy should ignore environment and TTY detection:

```go
var output bytes.Buffer
forceColor := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdout:         &output,
	ColorEnabled:   &forceColor,
	UnicodeEnabled: &unicode,
}))
console.Success("ANSI styling is forced")
fmt.Printf("%q\n", output.String())
// "\x1b[32m✔\x1b[0m ANSI styling is forced\n"
```

## What it provides

- Semantic action, information, success, warning, error, fatal, and debug messages.
- Plain printing and `io.Writer` adapters that coordinate with prompts and live displays.
- Automatic ANSI color policy with `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, and TTY awareness.
- ANSI-aware width, wrapping, end and middle truncation, indentation, and left/right/center padding.
- Printing and render-only forms of sections, rules, key/value summaries, lists, boxes, tables, and trees.
- Bordered tables by default, plus one compact form, fixed widths, alignment, wrapping, and ASCII fallbacks.
- Line-oriented questions, defaults, confirmation, numbered choices, and non-echoed secret input.
- Concurrency-safe loaders and determinate progress that become stable semantic lines when redirected.
- Configurable writers, input, marks, terminal hooks, environment lookup, and exit behavior.

## Design principles

- **Stay lightweight.** The only direct runtime dependency is `golang.org/x/term`.
- **Stay line-oriented.** Alternate screens, raw key events, cursor navigation, fuzzy selection, and multiple live widgets belong in a TUI package.
- **Keep output composable.** `Console` owns presentation policy, not structured logging, command routing, or application lifecycle.
- **Prefer durable logs outside a TTY.** Loaders and progress never leak carriage returns or erase sequences into redirected output.
- **Treat layout as terminal cells.** ANSI sequences are zero-width; combining marks, CJK text, and common emoji are measured for console alignment.
- **Offer one strong default.** Options cover common adjustments such as width, compact tables, and alignment without introducing themes or a styling matrix.
- **Make testing ordinary.** Writers, input, environment lookup, terminal detection, terminal size, and process exit are injectable.
- **Fail fast on invalid wiring.** `SetDefault(nil)` panics instead of leaving package helpers silently unusable.

## Layout

Layout helpers write through a `Console`. Their `RenderSection`, `RenderRule`, `RenderKeyValues`, `RenderList`, `RenderTree`, `RenderBox`, and `RenderTable` counterparts return the same presentation without a trailing newline, ready for composition.

```go
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	ColorEnabled:   &color,
	UnicodeEnabled: &unicode,
}))

fmt.Println(console.RenderBox(
	"All services healthy.",
	console.BoxTitle("Status"),
	console.BoxWidth(26),
))
// ┌─ Status ───────────────┐
// │ All services healthy.  │
// └────────────────────────┘
```

Tables are bordered and content-preserving by default: columns shrink and wrap when needed. `TableCompact`, `TableWidths`, `TableRightAlign`, and `TableCenterAlign` cover the common presentation changes without exposing a theme system or a large style vocabulary. Trees intentionally have no options beyond the console's Unicode/ASCII policy.

Borders, marks, list bullets, tree connectors, progress bars, and loader frames have ASCII fallbacks. Set `UnicodeEnabled` to `false` when targeting a constrained terminal; text measurement remains Unicode-aware.

## Loaders and progress

Constructing a loader or progress display has no side effects. `Start` claims the console's single transient line when animation is possible; another live loader or progress display receives `ErrTransientActive`.

```go
var output bytes.Buffer
color := false
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	Stdout:            &output,
	ColorEnabled:      &color,
	UnicodeEnabled:    &unicode,
	AnimationsEnabled: &animations,
}))

loader := console.NewLoader("Downloading modules")
if err := loader.Start(); err != nil {
	console.Error(err.Error())
	return
}

loader.Update("Verifying modules")
loader.Success("Modules ready")
fmt.Print(output.String())
// · Downloading modules
// ✔ Modules ready
```

On a terminal a loader animates in place, while progress uses one adaptive bar with a percentage. Narrow terminals fall back to message and percentage. Redirected output uses only a start line and the final success or error, so captured logs stay useful.

`Stop`, `Success`, `Warn`, and `Fail` are idempotent terminal operations; the first one wins. Complete output lines temporarily clear and redraw the active transient display. Partial `Print`/`Printf` output and prompts pause live output until the line completes or input returns.

Progress follows the same lifecycle: `Set`, `Add`, and `Update` change live state; `Complete`, `Fail`, or `Stop` finishes it, and the first terminal operation wins. Reaching the total does not implicitly report success.

## Prompts

Prompts refuse to read when the configured input and output are not terminals, returning `ErrNonInteractive` instead of unexpectedly blocking automation. Tests and intentional scripted input can opt in explicitly:

```go
var output bytes.Buffer
interactive := true
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("yes\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	ColorEnabled:       &color,
	UnicodeEnabled:     &unicode,
}))

confirmed, err := console.Confirm("Deploy now", false)
fmt.Printf("%q\n", output.String())
// "› Deploy now [y/N]: "
fmt.Println(confirmed, err)
// true <nil>
```

The prompt reader is retained for the lifetime of the console, so sequential prompts do not lose input to buffering.
While a prompt waits for input, complete writes from other goroutines wait rather than overwrite the live input line.

`AskSecret` uses terminal password input and never falls back to an echoed line. Tests and custom terminal integrations can inject `Config.ReadSecret`; the returned value is never printed by the package.

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
| Bordered and compact tables | `go -C examples run ./tables` |
| Redirect-safe loader lifecycle | `go -C examples run ./loader` |
| Redirect-safe progress lifecycle | `go -C examples run ./progress` |
| Scripted questions and confirmation | `go -C examples run ./prompts` |

<!-- api:embed:start -->

## API index

The complete API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/goforj/console).

| Group | API |
| --- | --- |
| Boxes | <a id="box"></a>[Box](https://pkg.go.dev/github.com/goforj/console#Box) · <a id="boxcolor"></a>[BoxColor](https://pkg.go.dev/github.com/goforj/console#BoxColor) · <a id="boxoption"></a>[BoxOption](https://pkg.go.dev/github.com/goforj/console#BoxOption) · <a id="boxpadding"></a>[BoxPadding](https://pkg.go.dev/github.com/goforj/console#BoxPadding) · <a id="boxtitle"></a>[BoxTitle](https://pkg.go.dev/github.com/goforj/console#BoxTitle) · <a id="boxwidth"></a>[BoxWidth](https://pkg.go.dev/github.com/goforj/console#BoxWidth) · <a id="console-box"></a>[Console.Box](https://pkg.go.dev/github.com/goforj/console#Console.Box) · <a id="console-renderbox"></a>[Console.RenderBox](https://pkg.go.dev/github.com/goforj/console#Console.RenderBox) · <a id="renderbox"></a>[RenderBox](https://pkg.go.dev/github.com/goforj/console#RenderBox) |
| Layout | <a id="console-keyvaluemap"></a>[Console.KeyValueMap](https://pkg.go.dev/github.com/goforj/console#Console.KeyValueMap) · <a id="console-keyvalues"></a>[Console.KeyValues](https://pkg.go.dev/github.com/goforj/console#Console.KeyValues) · <a id="console-list"></a>[Console.List](https://pkg.go.dev/github.com/goforj/console#Console.List) · <a id="console-numberedlist"></a>[Console.NumberedList](https://pkg.go.dev/github.com/goforj/console#Console.NumberedList) · <a id="console-renderkeyvaluemap"></a>[Console.RenderKeyValueMap](https://pkg.go.dev/github.com/goforj/console#Console.RenderKeyValueMap) · <a id="console-renderkeyvalues"></a>[Console.RenderKeyValues](https://pkg.go.dev/github.com/goforj/console#Console.RenderKeyValues) · <a id="console-renderlist"></a>[Console.RenderList](https://pkg.go.dev/github.com/goforj/console#Console.RenderList) · <a id="console-rendernumberedlist"></a>[Console.RenderNumberedList](https://pkg.go.dev/github.com/goforj/console#Console.RenderNumberedList) · <a id="console-renderrule"></a>[Console.RenderRule](https://pkg.go.dev/github.com/goforj/console#Console.RenderRule) · <a id="console-rendersection"></a>[Console.RenderSection](https://pkg.go.dev/github.com/goforj/console#Console.RenderSection) · <a id="console-rule"></a>[Console.Rule](https://pkg.go.dev/github.com/goforj/console#Console.Rule) · <a id="console-section"></a>[Console.Section](https://pkg.go.dev/github.com/goforj/console#Console.Section) · <a id="kv"></a>[KV](https://pkg.go.dev/github.com/goforj/console#KV) · <a id="keyvalue"></a>[KeyValue](https://pkg.go.dev/github.com/goforj/console#KeyValue) · <a id="keyvaluemap"></a>[KeyValueMap](https://pkg.go.dev/github.com/goforj/console#KeyValueMap) · <a id="keyvalues"></a>[KeyValues](https://pkg.go.dev/github.com/goforj/console#KeyValues) · <a id="list"></a>[List](https://pkg.go.dev/github.com/goforj/console#List) · <a id="numberedlist"></a>[NumberedList](https://pkg.go.dev/github.com/goforj/console#NumberedList) · <a id="renderkeyvaluemap"></a>[RenderKeyValueMap](https://pkg.go.dev/github.com/goforj/console#RenderKeyValueMap) · <a id="renderkeyvalues"></a>[RenderKeyValues](https://pkg.go.dev/github.com/goforj/console#RenderKeyValues) · <a id="renderlist"></a>[RenderList](https://pkg.go.dev/github.com/goforj/console#RenderList) · <a id="rendernumberedlist"></a>[RenderNumberedList](https://pkg.go.dev/github.com/goforj/console#RenderNumberedList) · <a id="renderrule"></a>[RenderRule](https://pkg.go.dev/github.com/goforj/console#RenderRule) · <a id="rendersection"></a>[RenderSection](https://pkg.go.dev/github.com/goforj/console#RenderSection) · <a id="rule"></a>[Rule](https://pkg.go.dev/github.com/goforj/console#Rule) · <a id="section"></a>[Section](https://pkg.go.dev/github.com/goforj/console#Section) |
| Loaders | <a id="console-loader"></a>[Console.Loader](https://pkg.go.dev/github.com/goforj/console#Console.Loader) · <a id="loader"></a>[Loader](https://pkg.go.dev/github.com/goforj/console#Loader) · <a id="loader-fail"></a>[Loader.Fail](https://pkg.go.dev/github.com/goforj/console#Loader.Fail) · <a id="loader-start"></a>[Loader.Start](https://pkg.go.dev/github.com/goforj/console#Loader.Start) · <a id="loader-stop"></a>[Loader.Stop](https://pkg.go.dev/github.com/goforj/console#Loader.Stop) · <a id="loader-success"></a>[Loader.Success](https://pkg.go.dev/github.com/goforj/console#Loader.Success) · <a id="loader-update"></a>[Loader.Update](https://pkg.go.dev/github.com/goforj/console#Loader.Update) · <a id="loader-warn"></a>[Loader.Warn](https://pkg.go.dev/github.com/goforj/console#Loader.Warn) · <a id="newloader"></a>[NewLoader](https://pkg.go.dev/github.com/goforj/console#NewLoader) |
| Marks | <a id="actionmark"></a>[ActionMark](https://pkg.go.dev/github.com/goforj/console#ActionMark) · <a id="console-actionmark"></a>[Console.ActionMark](https://pkg.go.dev/github.com/goforj/console#Console.ActionMark) · <a id="console-debugmark"></a>[Console.DebugMark](https://pkg.go.dev/github.com/goforj/console#Console.DebugMark) · <a id="console-errormark"></a>[Console.ErrorMark](https://pkg.go.dev/github.com/goforj/console#Console.ErrorMark) · <a id="console-infomark"></a>[Console.InfoMark](https://pkg.go.dev/github.com/goforj/console#Console.InfoMark) · <a id="console-successmark"></a>[Console.SuccessMark](https://pkg.go.dev/github.com/goforj/console#Console.SuccessMark) · <a id="console-warnmark"></a>[Console.WarnMark](https://pkg.go.dev/github.com/goforj/console#Console.WarnMark) · <a id="debugmark"></a>[DebugMark](https://pkg.go.dev/github.com/goforj/console#DebugMark) · <a id="errormark"></a>[ErrorMark](https://pkg.go.dev/github.com/goforj/console#ErrorMark) · <a id="infomark"></a>[InfoMark](https://pkg.go.dev/github.com/goforj/console#InfoMark) · <a id="successmark"></a>[SuccessMark](https://pkg.go.dev/github.com/goforj/console#SuccessMark) · <a id="warnmark"></a>[WarnMark](https://pkg.go.dev/github.com/goforj/console#WarnMark) |
| Messages | <a id="action"></a>[Action](https://pkg.go.dev/github.com/goforj/console#Action) · <a id="actionf"></a>[Actionf](https://pkg.go.dev/github.com/goforj/console#Actionf) · <a id="console-action"></a>[Console.Action](https://pkg.go.dev/github.com/goforj/console#Console.Action) · <a id="console-actionf"></a>[Console.Actionf](https://pkg.go.dev/github.com/goforj/console#Console.Actionf) · <a id="console-debug"></a>[Console.Debug](https://pkg.go.dev/github.com/goforj/console#Console.Debug) · <a id="console-debugf"></a>[Console.Debugf](https://pkg.go.dev/github.com/goforj/console#Console.Debugf) · <a id="console-error"></a>[Console.Error](https://pkg.go.dev/github.com/goforj/console#Console.Error) · <a id="console-errorf"></a>[Console.Errorf](https://pkg.go.dev/github.com/goforj/console#Console.Errorf) · <a id="console-fatal"></a>[Console.Fatal](https://pkg.go.dev/github.com/goforj/console#Console.Fatal) · <a id="console-fatalf"></a>[Console.Fatalf](https://pkg.go.dev/github.com/goforj/console#Console.Fatalf) · <a id="console-info"></a>[Console.Info](https://pkg.go.dev/github.com/goforj/console#Console.Info) · <a id="console-infof"></a>[Console.Infof](https://pkg.go.dev/github.com/goforj/console#Console.Infof) · <a id="console-success"></a>[Console.Success](https://pkg.go.dev/github.com/goforj/console#Console.Success) · <a id="console-successf"></a>[Console.Successf](https://pkg.go.dev/github.com/goforj/console#Console.Successf) · <a id="console-warn"></a>[Console.Warn](https://pkg.go.dev/github.com/goforj/console#Console.Warn) · <a id="console-warnf"></a>[Console.Warnf](https://pkg.go.dev/github.com/goforj/console#Console.Warnf) · <a id="debug"></a>[Debug](https://pkg.go.dev/github.com/goforj/console#Debug) · <a id="debugf"></a>[Debugf](https://pkg.go.dev/github.com/goforj/console#Debugf) · <a id="error"></a>[Error](https://pkg.go.dev/github.com/goforj/console#Error) · <a id="errorf"></a>[Errorf](https://pkg.go.dev/github.com/goforj/console#Errorf) · <a id="fatal"></a>[Fatal](https://pkg.go.dev/github.com/goforj/console#Fatal) · <a id="fatalf"></a>[Fatalf](https://pkg.go.dev/github.com/goforj/console#Fatalf) · <a id="info"></a>[Info](https://pkg.go.dev/github.com/goforj/console#Info) · <a id="infof"></a>[Infof](https://pkg.go.dev/github.com/goforj/console#Infof) · <a id="success"></a>[Success](https://pkg.go.dev/github.com/goforj/console#Success) · <a id="successf"></a>[Successf](https://pkg.go.dev/github.com/goforj/console#Successf) · <a id="warn"></a>[Warn](https://pkg.go.dev/github.com/goforj/console#Warn) · <a id="warnf"></a>[Warnf](https://pkg.go.dev/github.com/goforj/console#Warnf) |
| Output | <a id="console-newline"></a>[Console.NewLine](https://pkg.go.dev/github.com/goforj/console#Console.NewLine) · <a id="console-print"></a>[Console.Print](https://pkg.go.dev/github.com/goforj/console#Console.Print) · <a id="console-printf"></a>[Console.Printf](https://pkg.go.dev/github.com/goforj/console#Console.Printf) · <a id="console-println"></a>[Console.Println](https://pkg.go.dev/github.com/goforj/console#Console.Println) · <a id="console-stderrwriter"></a>[Console.StderrWriter](https://pkg.go.dev/github.com/goforj/console#Console.StderrWriter) · <a id="console-stdoutwriter"></a>[Console.StdoutWriter](https://pkg.go.dev/github.com/goforj/console#Console.StdoutWriter) · <a id="newline"></a>[NewLine](https://pkg.go.dev/github.com/goforj/console#NewLine) · <a id="print"></a>[Print](https://pkg.go.dev/github.com/goforj/console#Print) · <a id="printf"></a>[Printf](https://pkg.go.dev/github.com/goforj/console#Printf) · <a id="println"></a>[Println](https://pkg.go.dev/github.com/goforj/console#Println) · <a id="stderrwriter"></a>[StderrWriter](https://pkg.go.dev/github.com/goforj/console#StderrWriter) · <a id="stdoutwriter"></a>[StdoutWriter](https://pkg.go.dev/github.com/goforj/console#StdoutWriter) |
| Progress | <a id="console-progress"></a>[Console.Progress](https://pkg.go.dev/github.com/goforj/console#Console.Progress) · <a id="newprogress"></a>[NewProgress](https://pkg.go.dev/github.com/goforj/console#NewProgress) · <a id="progress"></a>[Progress](https://pkg.go.dev/github.com/goforj/console#Progress) · <a id="progress-add"></a>[Progress.Add](https://pkg.go.dev/github.com/goforj/console#Progress.Add) · <a id="progress-complete"></a>[Progress.Complete](https://pkg.go.dev/github.com/goforj/console#Progress.Complete) · <a id="progress-fail"></a>[Progress.Fail](https://pkg.go.dev/github.com/goforj/console#Progress.Fail) · <a id="progress-set"></a>[Progress.Set](https://pkg.go.dev/github.com/goforj/console#Progress.Set) · <a id="progress-start"></a>[Progress.Start](https://pkg.go.dev/github.com/goforj/console#Progress.Start) · <a id="progress-stop"></a>[Progress.Stop](https://pkg.go.dev/github.com/goforj/console#Progress.Stop) · <a id="progress-update"></a>[Progress.Update](https://pkg.go.dev/github.com/goforj/console#Progress.Update) |
| Prompts | <a id="ask"></a>[Ask](https://pkg.go.dev/github.com/goforj/console#Ask) · <a id="askdefault"></a>[AskDefault](https://pkg.go.dev/github.com/goforj/console#AskDefault) · <a id="asksecret"></a>[AskSecret](https://pkg.go.dev/github.com/goforj/console#AskSecret) · <a id="choose"></a>[Choose](https://pkg.go.dev/github.com/goforj/console#Choose) · <a id="confirm"></a>[Confirm](https://pkg.go.dev/github.com/goforj/console#Confirm) · <a id="console-ask"></a>[Console.Ask](https://pkg.go.dev/github.com/goforj/console#Console.Ask) · <a id="console-askdefault"></a>[Console.AskDefault](https://pkg.go.dev/github.com/goforj/console#Console.AskDefault) · <a id="console-asksecret"></a>[Console.AskSecret](https://pkg.go.dev/github.com/goforj/console#Console.AskSecret) · <a id="console-choose"></a>[Console.Choose](https://pkg.go.dev/github.com/goforj/console#Console.Choose) · <a id="console-confirm"></a>[Console.Confirm](https://pkg.go.dev/github.com/goforj/console#Console.Confirm) · <a id="errnoninteractive"></a>[ErrNonInteractive](https://pkg.go.dev/github.com/goforj/console#ErrNonInteractive) |
| Runtime | <a id="asciimarks"></a>[ASCIIMarks](https://pkg.go.dev/github.com/goforj/console#ASCIIMarks) · <a id="config"></a>[Config](https://pkg.go.dev/github.com/goforj/console#Config) · <a id="console"></a>[Console](https://pkg.go.dev/github.com/goforj/console#Console) · <a id="default"></a>[Default](https://pkg.go.dev/github.com/goforj/console#Default) · <a id="defaultmarks"></a>[DefaultMarks](https://pkg.go.dev/github.com/goforj/console#DefaultMarks) · <a id="marks"></a>[Marks](https://pkg.go.dev/github.com/goforj/console#Marks) · <a id="new"></a>[New](https://pkg.go.dev/github.com/goforj/console#New) · <a id="setdefault"></a>[SetDefault](https://pkg.go.dev/github.com/goforj/console#SetDefault) |
| Styling | <a id="colorblack"></a>[ColorBlack](https://pkg.go.dev/github.com/goforj/console#ColorBlack) · <a id="colorblue"></a>[ColorBlue](https://pkg.go.dev/github.com/goforj/console#ColorBlue) · <a id="colorboldgreen"></a>[ColorBoldGreen](https://pkg.go.dev/github.com/goforj/console#ColorBoldGreen) · <a id="colorboldwhite"></a>[ColorBoldWhite](https://pkg.go.dev/github.com/goforj/console#ColorBoldWhite) · <a id="colorcyan"></a>[ColorCyan](https://pkg.go.dev/github.com/goforj/console#ColorCyan) · <a id="colorgray"></a>[ColorGray](https://pkg.go.dev/github.com/goforj/console#ColorGray) · <a id="colorgreen"></a>[ColorGreen](https://pkg.go.dev/github.com/goforj/console#ColorGreen) · <a id="colormagenta"></a>[ColorMagenta](https://pkg.go.dev/github.com/goforj/console#ColorMagenta) · <a id="colorred"></a>[ColorRed](https://pkg.go.dev/github.com/goforj/console#ColorRed) · <a id="colorreset"></a>[ColorReset](https://pkg.go.dev/github.com/goforj/console#ColorReset) · <a id="colorwhite"></a>[ColorWhite](https://pkg.go.dev/github.com/goforj/console#ColorWhite) · <a id="coloryellow"></a>[ColorYellow](https://pkg.go.dev/github.com/goforj/console#ColorYellow) · <a id="colorize"></a>[Colorize](https://pkg.go.dev/github.com/goforj/console#Colorize) · <a id="console-colorize"></a>[Console.Colorize](https://pkg.go.dev/github.com/goforj/console#Console.Colorize) · <a id="console-style"></a>[Console.Style](https://pkg.go.dev/github.com/goforj/console#Console.Style) · <a id="style"></a>[Style](https://pkg.go.dev/github.com/goforj/console#Style) · <a id="stylebold"></a>[StyleBold](https://pkg.go.dev/github.com/goforj/console#StyleBold) · <a id="styledim"></a>[StyleDim](https://pkg.go.dev/github.com/goforj/console#StyleDim) · <a id="styleunderline"></a>[StyleUnderline](https://pkg.go.dev/github.com/goforj/console#StyleUnderline) |
| Tables | <a id="console-rendertable"></a>[Console.RenderTable](https://pkg.go.dev/github.com/goforj/console#Console.RenderTable) · <a id="console-table"></a>[Console.Table](https://pkg.go.dev/github.com/goforj/console#Console.Table) · <a id="rendertable"></a>[RenderTable](https://pkg.go.dev/github.com/goforj/console#RenderTable) · <a id="table"></a>[Table](https://pkg.go.dev/github.com/goforj/console#Table) · <a id="tablecenteralign"></a>[TableCenterAlign](https://pkg.go.dev/github.com/goforj/console#TableCenterAlign) · <a id="tablecompact"></a>[TableCompact](https://pkg.go.dev/github.com/goforj/console#TableCompact) · <a id="tableoption"></a>[TableOption](https://pkg.go.dev/github.com/goforj/console#TableOption) · <a id="tablerightalign"></a>[TableRightAlign](https://pkg.go.dev/github.com/goforj/console#TableRightAlign) · <a id="tablewidths"></a>[TableWidths](https://pkg.go.dev/github.com/goforj/console#TableWidths) |
| Terminal | <a id="console-isinteractive"></a>[Console.IsInteractive](https://pkg.go.dev/github.com/goforj/console#Console.IsInteractive) · <a id="console-supportscolor"></a>[Console.SupportsColor](https://pkg.go.dev/github.com/goforj/console#Console.SupportsColor) · <a id="console-supportsunicode"></a>[Console.SupportsUnicode](https://pkg.go.dev/github.com/goforj/console#Console.SupportsUnicode) · <a id="console-width"></a>[Console.Width](https://pkg.go.dev/github.com/goforj/console#Console.Width) · <a id="errtransientactive"></a>[ErrTransientActive](https://pkg.go.dev/github.com/goforj/console#ErrTransientActive) · <a id="isinteractive"></a>[IsInteractive](https://pkg.go.dev/github.com/goforj/console#IsInteractive) · <a id="supportscolor"></a>[SupportsColor](https://pkg.go.dev/github.com/goforj/console#SupportsColor) · <a id="supportsunicode"></a>[SupportsUnicode](https://pkg.go.dev/github.com/goforj/console#SupportsUnicode) · <a id="width"></a>[Width](https://pkg.go.dev/github.com/goforj/console#Width) |
| Text | <a id="console-expandtabs"></a>[Console.ExpandTabs](https://pkg.go.dev/github.com/goforj/console#Console.ExpandTabs) · <a id="console-indent"></a>[Console.Indent](https://pkg.go.dev/github.com/goforj/console#Console.Indent) · <a id="console-padcenter"></a>[Console.PadCenter](https://pkg.go.dev/github.com/goforj/console#Console.PadCenter) · <a id="console-padleft"></a>[Console.PadLeft](https://pkg.go.dev/github.com/goforj/console#Console.PadLeft) · <a id="console-padright"></a>[Console.PadRight](https://pkg.go.dev/github.com/goforj/console#Console.PadRight) · <a id="console-stripansi"></a>[Console.StripANSI](https://pkg.go.dev/github.com/goforj/console#Console.StripANSI) · <a id="console-truncate"></a>[Console.Truncate](https://pkg.go.dev/github.com/goforj/console#Console.Truncate) · <a id="console-truncatemiddle"></a>[Console.TruncateMiddle](https://pkg.go.dev/github.com/goforj/console#Console.TruncateMiddle) · <a id="console-visiblewidth"></a>[Console.VisibleWidth](https://pkg.go.dev/github.com/goforj/console#Console.VisibleWidth) · <a id="console-wrap"></a>[Console.Wrap](https://pkg.go.dev/github.com/goforj/console#Console.Wrap) · <a id="expandtabs"></a>[ExpandTabs](https://pkg.go.dev/github.com/goforj/console#ExpandTabs) · <a id="indent"></a>[Indent](https://pkg.go.dev/github.com/goforj/console#Indent) · <a id="padcenter"></a>[PadCenter](https://pkg.go.dev/github.com/goforj/console#PadCenter) · <a id="padleft"></a>[PadLeft](https://pkg.go.dev/github.com/goforj/console#PadLeft) · <a id="padright"></a>[PadRight](https://pkg.go.dev/github.com/goforj/console#PadRight) · <a id="stripansi"></a>[StripANSI](https://pkg.go.dev/github.com/goforj/console#StripANSI) · <a id="truncate"></a>[Truncate](https://pkg.go.dev/github.com/goforj/console#Truncate) · <a id="truncatemiddle"></a>[TruncateMiddle](https://pkg.go.dev/github.com/goforj/console#TruncateMiddle) · <a id="visiblewidth"></a>[VisibleWidth](https://pkg.go.dev/github.com/goforj/console#VisibleWidth) · <a id="wrap"></a>[Wrap](https://pkg.go.dev/github.com/goforj/console#Wrap) |
| Trees | <a id="console-rendertree"></a>[Console.RenderTree](https://pkg.go.dev/github.com/goforj/console#Console.RenderTree) · <a id="console-tree"></a>[Console.Tree](https://pkg.go.dev/github.com/goforj/console#Console.Tree) · <a id="node"></a>[Node](https://pkg.go.dev/github.com/goforj/console#Node) · <a id="rendertree"></a>[RenderTree](https://pkg.go.dev/github.com/goforj/console#RenderTree) · <a id="tree"></a>[Tree](https://pkg.go.dev/github.com/goforj/console#Tree) · <a id="treenode"></a>[TreeNode](https://pkg.go.dev/github.com/goforj/console#TreeNode) |

## Executable examples

These focused snippets are generated from standard Go example tests. The test suite executes each one and verifies every inline output comment.

### Semantic and multiline messages

```go
console.Action("Building application")
// · Building application
console.Success("API ready\nWorker ready")
// ✔ API ready
//   Worker ready
console.Warn("Configuration is incomplete")
// ! Configuration is incomplete
console.Error("Port already in use")
// ✖ Port already in use
```

### Plain output and coordinated writers

```go
console.Println("plain output")
// plain output
fmt.Fprintln(console.StdoutWriter(), "streamed output")
// streamed output
fmt.Fprintln(console.StderrWriter(), "diagnostic output")
// diagnostic output
```

### Adaptive styles and marks

```go
fmt.Println(console.ActionMark(), console.SuccessMark(), console.ErrorMark())
// · ✔ ✖
fmt.Println(console.Style("release ready", console.StyleBold, console.ColorGreen))
// release ready
```

### Sections, rules, and summaries

```go
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
```

### Bulleted and numbered lists

```go
console.List("validate configuration", "connect to database")
// • validate configuration
// • connect to database
console.NumberedList("build", "test", "publish")
// 1. build
// 2. test
// 3. publish
```

### Trees

```go
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
```

### Boxes

```go
console.Box(
	"The API and worker are healthy.",
	console.BoxTitle("Status"),
	console.BoxWidth(38),
)
// ┌─ Status ───────────────────────────┐
// │ The API and worker are healthy.    │
// └────────────────────────────────────┘
```

### Tables

```go
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
```

### Compact, fixed, and aligned tables

```go
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
```

### Redirect-safe loader outcomes

```go
download := console.NewLoader("Downloading modules")
_ = download.Start()
// · Downloading modules
download.Success("Modules ready")
// ✔ Modules ready

publish := console.NewLoader("Publishing release")
_ = publish.Start()
// · Publishing release
publish.Fail("Registry refused upload")
// ✖ Registry refused upload
```

### Determinate progress

```go
progress := console.NewProgress(100, "Packaging release")
_ = progress.Start()
// · Packaging release
progress.Set(40)
progress.Add(60)
progress.Complete("Release ready")
// ✔ Release ready
```

### Questions, defaults, and confirmation

```go
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
```

### Choices and secret input

```go
var output bytes.Buffer
interactive := true
color := false
unicode := true
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
secret, _ := console.AskSecret("API token")
fmt.Printf("%q\n", output.String())
// "Release channel\n1. stable\n2. beta\n› Choose [1-2, default 1]: \n› API token: \n"
fmt.Println(channel, len(secret))
// beta 11
```

### ANSI-aware text utilities

```go
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
```

### Isolated console instances

```go
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
```
<!-- api:embed:end -->

## Development

```sh
go test ./...
go test -race ./...
go -C docs test ./...
go -C examples test ./...
go generate .
go vet ./...
go -C docs vet ./...
go -C examples vet ./...
```

The docs and examples are separate Go modules so release archives contain only the library. The README API index is generated from public GoDoc and `@group` metadata; its representative examples and output come from standard Go example tests. Generation validates its marker pair before writing, so malformed hand-edited structure fails without partially changing the document.

## License

Released under the [MIT License](LICENSE).
