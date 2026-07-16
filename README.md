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

- **Stay lightweight.** The two focused direct dependencies are `golang.org/x/term` for terminal integration and `github.com/rivo/uniseg` for grapheme-aware cell measurement.
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
defer loader.Stop()

loader.Update("Verifying modules")
loader.Success("Modules ready")
fmt.Print(output.String())
// · Downloading modules
// ✔ Modules ready
```

On a terminal a loader animates in place, while progress uses one adaptive bar with a percentage. Narrow terminals fall back to message and percentage. With automatic policy, redirected output and conventional truthy `CI` environments use only a start line and the final success or error, so captured logs stay useful. `AnimationsEnabled` can explicitly opt a terminal back into animation.

Representative terminal snapshots are shown below; each display redraws one physical line rather than appending these frames. The last line is the exact Unicode rendering at a width of 14 cells.

```text
⠸ Verifying modules
Packaging release [██████████████░░░░░░░░░░]  60%
Packaging… 60%
```

`Stop`, `Success`, `Warn`, and `Fail` are idempotent terminal operations; the first one wins. Complete output lines temporarily clear and redraw the active transient display. Partial `Print`/`Printf` output and prompts pause live output until the line completes or input returns.

After `Start` succeeds, immediately `defer loader.Stop()` or `defer progress.Stop()`. An explicit success or failure wins first, while the deferred stop safely releases the transient line on an early return.

Progress follows the same lifecycle: `Set`, `Add`, and `Update` change live state, while `Step(current, message)` atomically sets the absolute completed amount and message. Use `Add` for relative increments. `Complete`, `Fail`, or `Stop` finishes it, and the first terminal operation wins. Reaching the total does not implicitly report success.

## Prompts

Prompts refuse to read when the configured input and output are not terminals—or when a conventional truthy `CI` value is present—returning `ErrNonInteractive` instead of unexpectedly blocking automation. Tests and intentional scripted input can opt in explicitly:

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

`Choose` returns the selected value. Use `ChooseIndex` when the zero-based position is more useful for looking up related configuration.

## Output behavior

Action, info, success, warning, and debug messages go to stdout. Error and fatal messages go to stderr. `Fatal` and `Fatalf` are the only helpers that exit; their exit function is configurable.
Marks follow the capability of their destination, so error styling is decided from stderr independently of stdout.

Color selection follows this precedence:

1. `Config.ColorEnabled` when set.
2. `NO_COLOR` disables styling.
3. A nonzero `CLICOLOR_FORCE` enables styling.
4. `CLICOLOR=0` or `TERM=dumb` disables styling.
5. Otherwise the destination must expose a terminal file descriptor with ANSI support.

On Windows, ANSI support is probed without leaving console mode changed. ANSI-bearing writes temporarily enable virtual-terminal processing and restore the original mode before returning.

Debug output is enabled by a nonzero `FORJ_DEBUG`, `APP_DEBUG`, or `DEBUG` value unless `Config.DebugEnabled` overrides it.

## Runnable examples

| Example | Command |
| --- | --- |
| Semantic messages and custom writers | `go -C examples run ./messages` |
| Boxes, key/value rows, and tables | `go -C examples run ./layout` |
| Bordered, compact, and ASCII tables | `go -C examples run ./tables` |
| Redirect-safe loader lifecycle | `go -C examples run ./loader` |
| Redirect-safe progress lifecycle | `go -C examples run ./progress` |
| Scripted questions and confirmation | `go -C examples run ./prompts` |
| Complete deployment lifecycle | `go -C examples run ./deploy` |
| Actionable validation report | `go -C examples run ./validation` |
| Machine stdout and status stderr | `go -C examples run ./ci` |

<!-- api:embed:start -->

## API index

Complete declaration documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/goforj/console). The links below open source-comment examples in this README. Package declarations and global helpers come first; `Console` methods provide the isolated equivalent, while loader and progress lifecycle methods remain on their returned values.

| Group | Package API | Instance and lifecycle API |
| --- | --- | --- |
| Boxes | [Box](#box) · [BoxColor](#boxcolor) · [BoxOption](#boxoption) · [BoxPadding](#boxpadding) · [BoxTitle](#boxtitle) · [BoxWidth](#boxwidth) · [RenderBox](#renderbox) | [Console.Box](#box) · [Console.RenderBox](#renderbox) |
| Layout | [KV](#kv) · [KeyValue](#keyvalue) · [KeyValueMap](#keyvaluemap) · [KeyValues](#keyvalues) · [List](#list) · [NumberedList](#numberedlist) · [RenderKeyValueMap](#renderkeyvaluemap) · [RenderKeyValues](#renderkeyvalues) · [RenderList](#renderlist) · [RenderNumberedList](#rendernumberedlist) · [RenderRule](#renderrule) · [RenderSection](#rendersection) · [Rule](#rule) · [Section](#section) | [Console.KeyValueMap](#keyvaluemap) · [Console.KeyValues](#keyvalues) · [Console.List](#list) · [Console.NumberedList](#numberedlist) · [Console.RenderKeyValueMap](#renderkeyvaluemap) · [Console.RenderKeyValues](#renderkeyvalues) · [Console.RenderList](#renderlist) · [Console.RenderNumberedList](#rendernumberedlist) · [Console.RenderRule](#renderrule) · [Console.RenderSection](#rendersection) · [Console.Rule](#rule) · [Console.Section](#section) |
| Loaders | [Loader](#loader) · [NewLoader](#newloader) | [Console.Loader](#newloader) · [Loader.Fail](#loader-fail) · [Loader.Start](#loader-start) · [Loader.Stop](#loader-stop) · [Loader.Success](#loader-success) · [Loader.Update](#loader-update) · [Loader.Warn](#loader-warn) |
| Marks | [ActionMark](#actionmark) · [DebugMark](#debugmark) · [ErrorMark](#errormark) · [InfoMark](#infomark) · [SuccessMark](#successmark) · [WarnMark](#warnmark) | [Console.ActionMark](#actionmark) · [Console.DebugMark](#debugmark) · [Console.ErrorMark](#errormark) · [Console.InfoMark](#infomark) · [Console.SuccessMark](#successmark) · [Console.WarnMark](#warnmark) |
| Messages | [Action](#action) · [Actionf](#actionf) · [Debug](#debug) · [Debugf](#debugf) · [Error](#error) · [Errorf](#errorf) · [Fatal](#fatal) · [Fatalf](#fatalf) · [Info](#info) · [Infof](#infof) · [Success](#success) · [Successf](#successf) · [Warn](#warn) · [Warnf](#warnf) | [Console.Action](#action) · [Console.Actionf](#actionf) · [Console.Debug](#debug) · [Console.Debugf](#debugf) · [Console.Error](#error) · [Console.Errorf](#errorf) · [Console.Fatal](#fatal) · [Console.Fatalf](#fatalf) · [Console.Info](#info) · [Console.Infof](#infof) · [Console.Success](#success) · [Console.Successf](#successf) · [Console.Warn](#warn) · [Console.Warnf](#warnf) |
| Output | [NewLine](#newline) · [Print](#print) · [Printf](#printf) · [Println](#println) · [StderrWriter](#stderrwriter) · [StdoutWriter](#stdoutwriter) | [Console.NewLine](#newline) · [Console.Print](#print) · [Console.Printf](#printf) · [Console.Println](#println) · [Console.StderrWriter](#stderrwriter) · [Console.StdoutWriter](#stdoutwriter) |
| Progress | [NewProgress](#newprogress) · [Progress](#progress) | [Console.Progress](#newprogress) · [Progress.Add](#progress-add) · [Progress.Complete](#progress-complete) · [Progress.Fail](#progress-fail) · [Progress.Set](#progress-set) · [Progress.Start](#progress-start) · [Progress.Step](#progress-step) · [Progress.Stop](#progress-stop) · [Progress.Update](#progress-update) |
| Prompts | [Ask](#ask) · [AskDefault](#askdefault) · [AskSecret](#asksecret) · [Choose](#choose) · [ChooseIndex](#chooseindex) · [Confirm](#confirm) · [ErrNonInteractive](#errnoninteractive) | [Console.Ask](#ask) · [Console.AskDefault](#askdefault) · [Console.AskSecret](#asksecret) · [Console.Choose](#choose) · [Console.ChooseIndex](#chooseindex) · [Console.Confirm](#confirm) |
| Runtime | [ASCIIMarks](#asciimarks) · [Config](#config) · [Console](#console) · [Default](#default) · [DefaultMarks](#defaultmarks) · [Marks](#marks) · [New](#new) · [SetDefault](#setdefault) | — |
| Styling | [ColorBlack](#colorblack) · [ColorBlue](#colorblue) · [ColorBoldGreen](#colorboldgreen) · [ColorBoldWhite](#colorboldwhite) · [ColorCyan](#colorcyan) · [ColorGray](#colorgray) · [ColorGreen](#colorgreen) · [ColorMagenta](#colormagenta) · [ColorRed](#colorred) · [ColorReset](#colorreset) · [ColorWhite](#colorwhite) · [ColorYellow](#coloryellow) · [Colorize](#colorize) · [Style](#style) · [StyleBold](#stylebold) · [StyleDim](#styledim) · [StyleUnderline](#styleunderline) | [Console.Colorize](#colorize) · [Console.Style](#style) |
| Tables | [RenderTable](#rendertable) · [Table](#table) · [TableCenterAlign](#tablecenteralign) · [TableCompact](#tablecompact) · [TableOption](#tableoption) · [TableRightAlign](#tablerightalign) · [TableWidths](#tablewidths) | [Console.RenderTable](#rendertable) · [Console.Table](#table) |
| Terminal | [ErrTransientActive](#errtransientactive) · [IsInteractive](#isinteractive) · [SupportsColor](#supportscolor) · [SupportsUnicode](#supportsunicode) · [Width](#width) | [Console.IsInteractive](#isinteractive) · [Console.SupportsColor](#supportscolor) · [Console.SupportsUnicode](#supportsunicode) · [Console.Width](#width) |
| Text | [ExpandTabs](#expandtabs) · [Indent](#indent) · [PadCenter](#padcenter) · [PadLeft](#padleft) · [PadRight](#padright) · [StripANSI](#stripansi) · [Truncate](#truncate) · [TruncateMiddle](#truncatemiddle) · [VisibleWidth](#visiblewidth) · [Wrap](#wrap) | [Console.ExpandTabs](#expandtabs) · [Console.Indent](#indent) · [Console.PadCenter](#padcenter) · [Console.PadLeft](#padleft) · [Console.PadRight](#padright) · [Console.StripANSI](#stripansi) · [Console.Truncate](#truncate) · [Console.TruncateMiddle](#truncatemiddle) · [Console.VisibleWidth](#visiblewidth) · [Console.Wrap](#wrap) |
| Trees | [Node](#node) · [RenderTree](#rendertree) · [Tree](#tree) · [TreeNode](#treenode) | [Console.RenderTree](#rendertree) · [Console.Tree](#tree) |

## API examples

These examples are generated from package GoDoc comments. Package-level helpers are shown in preference to equivalent `Console` methods.

### Boxes

#### <a id="box"></a>Box

Box prints a box through the default console.

```go
console.Box("ready", console.BoxTitle("Status"), console.BoxColor(""))
// ┌─ Status ┐
// │ ready   │
// └─────────┘
```

#### <a id="boxcolor"></a>BoxColor

BoxColor sets the ANSI color used for borders when styling is enabled.
An empty color leaves borders unstyled.

```go
fmt.Println(console.StripANSI(console.RenderBox("healthy", console.BoxColor(console.ColorGreen))))
// ┌─────────┐
// │ healthy │
// └─────────┘
```

#### <a id="boxoption"></a>BoxOption

BoxOption configures one rendered box.

```go
options := []console.BoxOption{
	console.BoxTitle("Status"),
	console.BoxColor(""),
}
fmt.Println(console.RenderBox("ready", options...))
// ┌─ Status ┐
// │ ready   │
// └─────────┘
```

#### <a id="boxpadding"></a>BoxPadding

BoxPadding sets the horizontal padding on both sides of the content.
Negative values are treated as zero, and padding is capped when necessary to fit the console width.

```go
fmt.Println(console.RenderBox("ready", console.BoxPadding(0), console.BoxColor("")))
// ┌─────┐
// │ready│
// └─────┘
```

#### <a id="boxtitle"></a>BoxTitle

BoxTitle adds a title to the top border.

```go
fmt.Println(console.RenderBox("ready", console.BoxTitle("Status"), console.BoxColor("")))
// ┌─ Status ┐
// │ ready   │
// └─────────┘
```

#### <a id="boxwidth"></a>BoxWidth

BoxWidth fixes the total visible width, including borders and padding.
Values below the structural minimum expand enough to preserve a valid frame.
Values less than one select an automatic width bounded by the console width.
Larger values are bounded by the console width when the structural minimum permits.

```go
fmt.Println(console.RenderBox("ready", console.BoxWidth(16), console.BoxColor("")))
// ┌──────────────┐
// │ ready        │
// └──────────────┘
```

#### <a id="renderbox"></a>RenderBox

RenderBox renders a box using the default console.

```go
fmt.Println(console.RenderBox("complete", console.BoxColor("")))
// ┌──────────┐
// │ complete │
// └──────────┘
```


### Layout

#### <a id="kv"></a>KV

KV creates one ordered key/value entry.

```go
console.KeyValues(console.KV("Region", "eu-west-1"))
// Region  eu-west-1
```

#### <a id="keyvalue"></a>KeyValue

KeyValue contains one ordered label and value for KeyValues.

```go
entries := []console.KeyValue{
	{Key: "Mode", Value: "production"},
	{Key: "Port", Value: 8080},
}
console.KeyValues(entries...)
// Mode  production
// Port  8080
```

#### <a id="keyvaluemap"></a>KeyValueMap

KeyValueMap prints a sorted key/value map through the default console.

```go
console.KeyValueMap(map[string]any{"port": 8080, "mode": "production"})
// mode  production
// port  8080
```

#### <a id="keyvalues"></a>KeyValues

KeyValues prints ordered key/value entries through the default console.

```go
console.KeyValues(
	console.KV("Mode", "production"),
	console.KV("Port", 8080),
)
// Mode  production
// Port  8080
```

#### <a id="list"></a>List

List prints an unordered list through the default console.

```go
console.List("build", "test", "publish")
// • build
// • test
// • publish
```

#### <a id="numberedlist"></a>NumberedList

NumberedList prints an ordered list through the default console.

```go
console.NumberedList("build", "test", "publish")
// 1. build
// 2. test
// 3. publish
```

#### <a id="renderkeyvaluemap"></a>RenderKeyValueMap

RenderKeyValueMap renders a sorted key/value map through the default console.

```go
fmt.Println(console.RenderKeyValueMap(map[string]any{"port": 8080, "mode": "production"}))
// mode  production
// port  8080
```

#### <a id="renderkeyvalues"></a>RenderKeyValues

RenderKeyValues renders ordered key/value entries through the default console.

```go
fmt.Println(console.RenderKeyValues(
	console.KV("Mode", "production"),
	console.KV("Port", 8080),
))
// Mode  production
// Port  8080
```

#### <a id="renderlist"></a>RenderList

RenderList renders an unordered list through the default console.

```go
fmt.Println(console.RenderList("build", "test"))
// • build
// • test
```

#### <a id="rendernumberedlist"></a>RenderNumberedList

RenderNumberedList renders an ordered list through the default console.

```go
fmt.Println(console.RenderNumberedList("build", "test"))
// 1. build
// 2. test
```

#### <a id="renderrule"></a>RenderRule

RenderRule renders a horizontal rule through the default console.

```go
previous := console.Default()
defer console.SetDefault(previous)
console.SetDefault(console.New(console.Config{Width: 16}))
fmt.Println(console.RenderRule("Next"))
// ── Next ────────
```

#### <a id="rendersection"></a>RenderSection

RenderSection renders a section heading through the default console.

```go
fmt.Println(console.RenderSection("Deployment"))
// ◇ Deployment
```

#### <a id="rule"></a>Rule

Rule prints a horizontal rule through the default console.

```go
previous := console.Default()
defer console.SetDefault(previous)
console.SetDefault(console.New(console.Config{Width: 16}))
console.Rule("Next")
// ── Next ────────
```

#### <a id="section"></a>Section

Section prints a section heading through the default console.

```go
console.Section("Deployment")
// ◇ Deployment
```


### Loaders

#### <a id="loader"></a>Loader

Loader presents one transient activity line on terminals and stable semantic lines in redirected output.
A Loader is concurrency-safe, single-use, and must be constructed with Console.Loader or NewLoader;
the first call to Stop, Success, Warn, or Fail wins.

```go
animations := false
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	ColorEnabled:      &color,
	UnicodeEnabled:    &unicode,
}))
var loader *console.Loader = console.NewLoader("Building application")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Building application
loader.Success("Application ready")
// ✔ Application ready
```

#### <a id="newloader"></a>NewLoader

NewLoader constructs a loader using a snapshot of the current default console.
It does not start the loader.

```go
animations := false
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	ColorEnabled:      &color,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Downloading modules")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Downloading modules
loader.Success("Modules ready")
// ✔ Modules ready
```

#### <a id="loader-fail"></a>Loader.Fail

Fail completes the loader with an error message on stderr.
An empty message reuses the loader's current message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Uploading release")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Uploading release
loader.Fail("Registry refused upload")
// ✖ Registry refused upload
```

#### <a id="loader-start"></a>Loader.Start

Start begins the loader and is harmless when called more than once.
Animated loaders can return ErrTransientActive when another live display owns the same console.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Building application")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Building application
loader.Stop()
```

#### <a id="loader-stop"></a>Loader.Stop

Stop removes the transient loader without printing a completion message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Checking configuration")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Checking configuration
loader.Stop()
```

#### <a id="loader-success"></a>Loader.Success

Success completes the loader with a success message.
An empty message reuses the loader's current message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Publishing release")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Publishing release
loader.Success("Release published")
// ✔ Release published
```

#### <a id="loader-update"></a>Loader.Update

Update changes the loader message and immediately redraws an active animation.
Updates after a terminal operation are ignored.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Downloading modules")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Downloading modules
loader.Update("Verifying modules")
loader.Success("")
// ✔ Verifying modules
```

#### <a id="loader-warn"></a>Loader.Warn

Warn completes the loader with a warning message.
An empty message reuses the loader's current message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
loader := console.NewLoader("Checking optional tools")
if err := loader.Start(); err != nil {
	panic(err)
}
// · Checking optional tools
loader.Warn("Optional tool not found")
// ! Optional tool not found
```


### Marks

#### <a id="actionmark"></a>ActionMark

ActionMark returns the default console's action indicator.

```go
fmt.Println(console.ActionMark())
// ·
```

#### <a id="debugmark"></a>DebugMark

DebugMark returns the default console's debug indicator.

```go
fmt.Println(console.DebugMark())
// ?
```

#### <a id="errormark"></a>ErrorMark

ErrorMark returns the default console's error indicator.

```go
fmt.Println(console.ErrorMark())
// ✖
```

#### <a id="infomark"></a>InfoMark

InfoMark returns the default console's informational indicator.

```go
fmt.Println(console.InfoMark())
// ·
```

#### <a id="successmark"></a>SuccessMark

SuccessMark returns the default console's success indicator.

```go
fmt.Println(console.SuccessMark())
// ✔
```

#### <a id="warnmark"></a>WarnMark

WarnMark returns the default console's warning indicator.

```go
fmt.Println(console.WarnMark())
// !
```


### Messages

#### <a id="action"></a>Action

Action prints an action message through the default console.

```go
console.Action("building release")
// · building release
```

#### <a id="actionf"></a>Actionf

Actionf prints a formatted action message through the default console.

```go
console.Actionf("building %s", "release")
// · building release
```

#### <a id="debug"></a>Debug

Debug prints a diagnostic message through the default console when enabled.

```go
debug := true
console.SetDefault(console.New(console.Config{DebugEnabled: &debug}))
console.Debug("cache miss")
// ? cache miss
```

#### <a id="debugf"></a>Debugf

Debugf prints a formatted diagnostic message through the default console when enabled.

```go
debug := true
console.SetDefault(console.New(console.Config{DebugEnabled: &debug}))
console.Debugf("attempt %d of %d", 1, 3)
// ? attempt 1 of 3
```

#### <a id="error"></a>Error

Error prints an error message through the default console.

```go
console.Error("deployment failed")
// ✖ deployment failed
```

#### <a id="errorf"></a>Errorf

Errorf prints a formatted error message through the default console.

```go
console.Errorf("deployment failed: %s", "timeout")
// ✖ deployment failed: timeout
```

#### <a id="fatal"></a>Fatal

Fatal prints an error through the default console and exits with status 1.

```go
console.SetDefault(console.New(console.Config{
	Exit: func(code int) { fmt.Println("exit", code) },
}))
console.Fatal("invalid configuration")
// ✖ invalid configuration
// exit 1
```

#### <a id="fatalf"></a>Fatalf

Fatalf prints a formatted error through the default console and exits with status 1.

```go
console.SetDefault(console.New(console.Config{
	Exit: func(code int) { fmt.Println("exit", code) },
}))
console.Fatalf("invalid port: %d", 0)
// ✖ invalid port: 0
// exit 1
```

#### <a id="info"></a>Info

Info prints an informational message through the default console.

```go
console.Info("using cached dependencies")
// · using cached dependencies
```

#### <a id="infof"></a>Infof

Infof prints a formatted informational message through the default console.

```go
console.Infof("using %s dependencies", "cached")
// · using cached dependencies
```

#### <a id="success"></a>Success

Success prints a success message through the default console.

```go
console.Success("release published")
// ✔ release published
```

#### <a id="successf"></a>Successf

Successf prints a formatted success message through the default console.

```go
console.Successf("published %s", "v1.2.0")
// ✔ published v1.2.0
```

#### <a id="warn"></a>Warn

Warn prints a warning message through the default console.

```go
console.Warn("configuration is deprecated")
// ! configuration is deprecated
```

#### <a id="warnf"></a>Warnf

Warnf prints a formatted warning message through the default console.

```go
console.Warnf("retrying in %d seconds", 5)
// ! retrying in 5 seconds
```


### Output

#### <a id="newline"></a>NewLine

NewLine writes one blank line through the default console.

```go
console.Println("before")
console.NewLine()
console.Println("after")
// before
//
// after
```

#### <a id="print"></a>Print

Print writes values through the default console without adding a newline.

```go
var output bytes.Buffer
console.SetDefault(console.New(console.Config{Stdout: &output}))
console.Print("deploying")
fmt.Printf("%q\n", output.String())
// "deploying"
```

#### <a id="printf"></a>Printf

Printf writes formatted output through the default console without adding a newline.

```go
var output bytes.Buffer
console.SetDefault(console.New(console.Config{Stdout: &output}))
console.Printf("copied %d files", 3)
fmt.Printf("%q\n", output.String())
// "copied 3 files"
```

#### <a id="println"></a>Println

Println writes values through the default console followed by a newline.

```go
console.Println("deployment complete")
// deployment complete
```

#### <a id="stderrwriter"></a>StderrWriter

StderrWriter returns a coordinated writer using a snapshot of the current default console.
Later calls to SetDefault do not retarget an existing writer.

```go
fmt.Fprintln(console.StderrWriter(), "download failed")
// download failed
```

#### <a id="stdoutwriter"></a>StdoutWriter

StdoutWriter returns a coordinated writer using a snapshot of the current default console.
Later calls to SetDefault do not retarget an existing writer.

```go
fmt.Fprintln(console.StdoutWriter(), "download complete")
// download complete
```


### Progress

#### <a id="newprogress"></a>NewProgress

NewProgress constructs a progress display using a snapshot of the current default console.
It does not start the display.

```go
animations := false
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	ColorEnabled:      &color,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(2, "Deploying services")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Deploying services
progress.Add(2)
progress.Complete("Services deployed")
// ✔ Services deployed
```

#### <a id="progress"></a>Progress

Progress presents determinate work as one transient terminal line and stable semantic lines in redirected output.
A Progress is concurrency-safe, single-use, and must be constructed with Console.Progress or NewProgress;
the first call to Complete, Fail, or Stop wins.

```go
animations := false
color := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	ColorEnabled:      &color,
	UnicodeEnabled:    &unicode,
}))
var progress *console.Progress = console.NewProgress(2, "Deploying services")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Deploying services
progress.Add(2)
progress.Complete("Services deployed")
// ✔ Services deployed
```

#### <a id="progress-add"></a>Progress.Add

Add changes the completed amount by delta and clamps it between zero and the total.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(2, "Deploying services")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Deploying services
progress.Add(1)
progress.Add(1)
progress.Complete("Services deployed")
// ✔ Services deployed
```

#### <a id="progress-complete"></a>Progress.Complete

Complete fills and finishes the display with a success message.
An empty message reuses the current progress message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(1, "Packaging release")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Packaging release
progress.Complete("Release ready")
// ✔ Release ready
```

#### <a id="progress-fail"></a>Progress.Fail

Fail finishes the display with an error message on stderr.
An empty message reuses the current progress message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(1, "Publishing release")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Publishing release
progress.Fail("Registry refused upload")
// ✖ Registry refused upload
```

#### <a id="progress-set"></a>Progress.Set

Set replaces the completed amount and clamps it between zero and the total.
Reaching the total does not complete the display; Complete records the durable outcome.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(100, "Uploading release")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Uploading release
progress.Set(100)
progress.Complete("Release uploaded")
// ✔ Release uploaded
```

#### <a id="progress-start"></a>Progress.Start

Start begins the progress display and is harmless when called more than once.
Live terminal displays can return ErrTransientActive when another display owns the console.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(3, "Running checks")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Running checks
progress.Stop()
```

#### <a id="progress-step"></a>Progress.Step

Step replaces the completed amount and message in one atomic progress update.
The amount is clamped between zero and the total, and updates after a terminal
operation are ignored.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(2, "Deploying API")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Deploying API
progress.Step(1, "Deploying worker")
progress.Complete("")
// ✔ Deploying worker
```

#### <a id="progress-stop"></a>Progress.Stop

Stop removes the transient display without printing a completion message.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(1, "Checking release")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Checking release
progress.Stop()
```

#### <a id="progress-update"></a>Progress.Update

Update changes the progress message and immediately redraws a live terminal display.
Updates after a terminal operation are ignored.

```go
animations := false
unicode := true
console.SetDefault(console.New(console.Config{
	AnimationsEnabled: &animations,
	UnicodeEnabled:    &unicode,
}))
progress := console.NewProgress(2, "Deploying API")
if err := progress.Start(); err != nil {
	panic(err)
}
// · Deploying API
progress.Update("Deploying worker")
progress.Complete("")
// ✔ Deploying worker
```


### Prompts

#### <a id="ask"></a>Ask

Ask prompts through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("Ada\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
}))
name, err := console.Ask("Name")
fmt.Printf("%q\n", output.String())
// "› Name: "
fmt.Println(name, err)
// Ada <nil>
```

#### <a id="askdefault"></a>AskDefault

AskDefault prompts with a default through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
}))
environment, err := console.AskDefault("Environment", "production")
fmt.Printf("%q\n", output.String())
// "› Environment [production]: "
fmt.Println(environment, err)
// production <nil>
```

#### <a id="asksecret"></a>AskSecret

AskSecret prompts without echoing input through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
	ReadSecret: func() (string, error) {
		return "token-value", nil
	},
}))
secret, err := console.AskSecret("API token")
fmt.Printf("%q\n", output.String())
// "› API token: \n"
fmt.Println(len(secret), err)
// 11 <nil>
```

#### <a id="choose"></a>Choose

Choose asks the user to select an option through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("2\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
}))
channel, err := console.Choose("Release channel", []string{"stable", "beta"}, 0)
fmt.Printf("%q\n", output.String())
// "Release channel\n1. stable\n2. beta\n› Choose [1-2, default 1]: "
fmt.Println(channel, err)
// beta <nil>
```

#### <a id="chooseindex"></a>ChooseIndex

ChooseIndex asks the user to select an option index through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("2\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
}))
index, err := console.ChooseIndex("Release channel", []string{"stable", "beta"}, 0)
fmt.Printf("%q\n", output.String())
// "Release channel\n1. stable\n2. beta\n› Choose [1-2, default 1]: "
fmt.Println(index, err)
// 1 <nil>
```

#### <a id="confirm"></a>Confirm

Confirm asks for confirmation through the default console.

```go
var output bytes.Buffer
interactive := true
unicode := true
console.SetDefault(console.New(console.Config{
	Stdin:              strings.NewReader("yes\n"),
	Stdout:             &output,
	InteractiveEnabled: &interactive,
	UnicodeEnabled:     &unicode,
}))
confirmed, err := console.Confirm("Deploy now", false)
fmt.Printf("%q\n", output.String())
// "› Deploy now [y/N]: "
fmt.Println(confirmed, err)
// true <nil>
```

#### <a id="errnoninteractive"></a>ErrNonInteractive

ErrNonInteractive is returned when a prompt would read from a console that is not interactive.
Set Config.InteractiveEnabled when intentionally driving prompts with an injected reader.

```go
interactive := false
console.SetDefault(console.New(console.Config{InteractiveEnabled: &interactive}))
_, err := console.Ask("Name")
fmt.Println(errors.Is(err, console.ErrNonInteractive))
// true
```


### Runtime

#### <a id="asciimarks"></a>ASCIIMarks

ASCIIMarks returns symbols suitable for constrained terminals and plain logs.

```go
marks := console.ASCIIMarks()
fmt.Println(marks.Success, marks.Warn, marks.Error)
// + ! x
```

#### <a id="config"></a>Config

Config configures a Console instance.

Every field is optional. Nil functions and writers use their operating-system
defaults, while nil boolean pointers select automatic behavior.

```go
configuration := console.Config{Width: 100}
commandConsole := console.New(configuration)
fmt.Println(commandConsole.Width())
// 100
```

#### <a id="console"></a>Console

Console coordinates output policy, terminal capabilities, prompts, and transient displays.
A Console is safe for concurrent message writes and must be constructed with New.

```go
var commandConsole *console.Console = console.New(console.Config{Width: 120})
fmt.Println(commandConsole.Width())
// 120
```

#### <a id="default"></a>Default

Default returns the console currently used by package-level helpers.

```go
fmt.Println(console.Default() != nil)
// true
```

#### <a id="defaultmarks"></a>DefaultMarks

DefaultMarks returns the Unicode symbols used by a default console.

```go
marks := console.DefaultMarks()
fmt.Println(marks.Success, marks.Warn, marks.Error)
// ✔ ! ✖
```

#### <a id="marks"></a>Marks

Marks contains the symbols used for messages, lists, selections, and loaders.

```go
marks := console.Marks{Success: "OK"}
fmt.Println(marks.Success)
// OK
```

#### <a id="new"></a>New

New creates an isolated console with optional runtime overrides.

```go
var output bytes.Buffer
color := false
unicode := true
commandConsole := console.New(console.Config{
	Stdout:         &output,
	ColorEnabled:   &color,
	UnicodeEnabled: &unicode,
})
commandConsole.Success("ready")
fmt.Print(output.String())
// ✔ ready
```

#### <a id="setdefault"></a>SetDefault

SetDefault replaces the console used by package-level helpers.
It panics when console is nil because package helpers always require a usable runtime.

```go
previous := console.Default()
defer console.SetDefault(previous)

var output bytes.Buffer
console.SetDefault(console.New(console.Config{Stdout: &output}))
console.Println("ready")
fmt.Print(output.String())
// ready
```


### Styling

#### <a id="colorblack"></a>ColorBlack

ColorBlack is a black ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorBlack)
// "\x1b[30m"
```

#### <a id="colorblue"></a>ColorBlue

ColorBlue is a blue ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorBlue)
// "\x1b[34m"
```

#### <a id="colorboldgreen"></a>ColorBoldGreen

ColorBoldGreen is a bold green ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorBoldGreen)
// "\x1b[1;32m"
```

#### <a id="colorboldwhite"></a>ColorBoldWhite

ColorBoldWhite is a bold white ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorBoldWhite)
// "\x1b[1;97m"
```

#### <a id="colorcyan"></a>ColorCyan

ColorCyan is a cyan ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorCyan)
// "\x1b[36m"
```

#### <a id="colorgray"></a>ColorGray

ColorGray is a muted gray ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorGray)
// "\x1b[90m"
```

#### <a id="colorgreen"></a>ColorGreen

ColorGreen is a green ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorGreen)
// "\x1b[32m"
```

#### <a id="colormagenta"></a>ColorMagenta

ColorMagenta is a magenta ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorMagenta)
// "\x1b[35m"
```

#### <a id="colorred"></a>ColorRed

ColorRed is a red ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorRed)
// "\x1b[31m"
```

#### <a id="colorreset"></a>ColorReset

ColorReset resets ANSI styling.

```go
fmt.Printf("%q\n", console.ColorReset)
// "\x1b[0m"
```

#### <a id="colorwhite"></a>ColorWhite

ColorWhite is a white ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorWhite)
// "\x1b[37m"
```

#### <a id="coloryellow"></a>ColorYellow

ColorYellow is a yellow ANSI foreground color.

```go
fmt.Printf("%q\n", console.ColorYellow)
// "\x1b[33m"
```

#### <a id="colorize"></a>Colorize

Colorize applies an ANSI color using the default console's color policy.

```go
color := true
console.SetDefault(console.New(console.Config{ColorEnabled: &color}))
fmt.Printf("%q\n", console.Colorize(console.ColorCyan, "connected"))
// "\x1b[36mconnected\x1b[0m"
```

#### <a id="style"></a>Style

Style applies ANSI styles using the default console's color policy.

```go
color := true
console.SetDefault(console.New(console.Config{ColorEnabled: &color}))
fmt.Printf("%q\n", console.Style("ready", console.StyleBold, console.ColorGreen))
// "\x1b[1m\x1b[32mready\x1b[0m"
```

#### <a id="stylebold"></a>StyleBold

StyleBold enables bold ANSI text.

```go
fmt.Printf("%q\n", console.StyleBold)
// "\x1b[1m"
```

#### <a id="styledim"></a>StyleDim

StyleDim enables dim ANSI text.

```go
fmt.Printf("%q\n", console.StyleDim)
// "\x1b[2m"
```

#### <a id="styleunderline"></a>StyleUnderline

StyleUnderline enables underlined ANSI text.

```go
fmt.Printf("%q\n", console.StyleUnderline)
// "\x1b[4m"
```


### Tables

#### <a id="rendertable"></a>RenderTable

RenderTable renders a table using the default console.

```go
fmt.Println(console.RenderTable(
	[]string{"Name", "State"},
	[][]string{{"worker", "idle"}},
))
// ┌────────┬───────┐
// │ Name   │ State │
// ├────────┼───────┤
// │ worker │ idle  │
// └────────┴───────┘
```

#### <a id="table"></a>Table

Table prints a table through the default console.

```go
console.Table(
	[]string{"Name", "State"},
	[][]string{{"api", "ready"}},
)
// ┌──────┬───────┐
// │ Name │ State │
// ├──────┼───────┤
// │ api  │ ready │
// └──────┴───────┘
```

#### <a id="tablecenteralign"></a>TableCenterAlign

TableCenterAlign centers the headers and values in the selected zero-based columns.
Negative and out-of-range columns are ignored.

```go
fmt.Println(console.RenderTable(
	[]string{"Service", "State"},
	[][]string{{"api", "up"}},
	console.TableCenterAlign(1),
))
// ┌─────────┬───────┐
// │ Service │ State │
// ├─────────┼───────┤
// │ api     │  up   │
// └─────────┴───────┘
```

#### <a id="tablecompact"></a>TableCompact

TableCompact removes the outer frame and separates columns with two spaces.
A compact table with headers retains one horizontal separator for readability.

```go
fmt.Println(console.RenderTable(
	[]string{"Name", "State"},
	[][]string{{"api", "ready"}},
	console.TableCompact(),
))
// Name  State
// ────  ─────
// api   ready
```

#### <a id="tableoption"></a>TableOption

TableOption configures one rendered table.

```go
options := []console.TableOption{console.TableCompact()}
fmt.Println(console.RenderTable(
	[]string{"Name", "State"},
	[][]string{{"api", "ready"}},
	options...,
))
// Name  State
// ────  ─────
// api   ready
```

#### <a id="tablerightalign"></a>TableRightAlign

TableRightAlign right-aligns the headers and values in the selected zero-based columns.
Negative and out-of-range columns are ignored.

```go
fmt.Println(console.RenderTable(
	[]string{"Item", "Count"},
	[][]string{{"api", "12"}},
	console.TableRightAlign(1),
))
// ┌──────┬───────┐
// │ Item │ Count │
// ├──────┼───────┤
// │ api  │    12 │
// └──────┴───────┘
```

#### <a id="tablewidths"></a>TableWidths

TableWidths sets content widths by zero-based column position.
Values less than one leave that column automatic, and configured widths may
still shrink when the complete table would exceed the console width.

```go
fmt.Println(console.RenderTable(
	[]string{"Name", "State"},
	[][]string{{"api", "ready"}},
	console.TableWidths(6, 7),
))
// ┌────────┬─────────┐
// │ Name   │ State   │
// ├────────┼─────────┤
// │ api    │ ready   │
// └────────┴─────────┘
```


### Terminal

#### <a id="errtransientactive"></a>ErrTransientActive

ErrTransientActive is returned when another live loader or progress display owns the transient line.

```go
fmt.Println(console.ErrTransientActive)
// console: another transient display is already active
```

#### <a id="isinteractive"></a>IsInteractive

IsInteractive reports whether the default console is interactive.

```go
previous := console.Default()
defer console.SetDefault(previous)
interactive := true
console.SetDefault(console.New(console.Config{InteractiveEnabled: &interactive}))

fmt.Println(console.IsInteractive())
// true
```

#### <a id="supportscolor"></a>SupportsColor

SupportsColor reports whether the default console emits ANSI styling.

```go
previous := console.Default()
defer console.SetDefault(previous)
color := true
console.SetDefault(console.New(console.Config{ColorEnabled: &color}))

fmt.Println(console.SupportsColor())
// true
```

#### <a id="supportsunicode"></a>SupportsUnicode

SupportsUnicode reports whether the default console uses Unicode presentation characters.

```go
previous := console.Default()
defer console.SetDefault(previous)
unicode := false
console.SetDefault(console.New(console.Config{UnicodeEnabled: &unicode}))

fmt.Println(console.SupportsUnicode())
// false
```

#### <a id="width"></a>Width

Width returns the width of the default console.

```go
previous := console.Default()
defer console.SetDefault(previous)
console.SetDefault(console.New(console.Config{Width: 100}))

fmt.Println(console.Width())
// 100
```


### Text

#### <a id="expandtabs"></a>ExpandTabs

ExpandTabs replaces tabs with spaces at eight-cell stops on each line.
ANSI escape sequences do not affect tab positions.

```go
fmt.Printf("%q\n", console.ExpandTabs("a\tb"))
// "a       b"
```

#### <a id="indent"></a>Indent

Indent prefixes every line in value with prefix.
Empty input remains empty.

```go
fmt.Printf("%q\n", console.Indent("one\ntwo", "> "))
// "> one\n> two"
```

#### <a id="padcenter"></a>PadCenter

PadCenter adds spaces around every line until it reaches width terminal cells.
Odd padding places the extra space on the right. Lines already at or beyond width are unchanged.
Tabs are expanded only on lines that need padding so their alignment remains stable.

```go
fmt.Printf("%q\n", console.PadCenter("go", 6))
// "  go  "
```

#### <a id="padleft"></a>PadLeft

PadLeft prepends spaces until every line reaches width terminal cells.
Lines already at or beyond width are unchanged. Tabs are expanded only on lines that need padding
because leading spaces otherwise change their terminal tab stops.

```go
fmt.Printf("%q\n", console.PadLeft("go", 5))
// "   go"
```

#### <a id="padright"></a>PadRight

PadRight appends spaces until every line reaches width terminal cells.
Lines already at or beyond width are unchanged.

```go
fmt.Printf("%q\n", console.PadRight("go", 5))
// "go   "
```

#### <a id="stripansi"></a>StripANSI

StripANSI removes complete ANSI CSI, OSC, and ESC sequences from value.
Incomplete escape sequences are retained so malformed input is not silently discarded.

```go
fmt.Println(console.StripANSI(console.ColorRed + "failed" + console.ColorReset))
// failed
```

#### <a id="truncate"></a>Truncate

Truncate shortens each line of value to width terminal cells and uses an ellipsis when content is removed.
Active SGR styles and OSC 8 hyperlinks are closed before the ellipsis.
Values less than one produce an empty string.

```go
fmt.Println(console.Truncate("deployment", 7))
// deploy…
```

#### <a id="truncatemiddle"></a>TruncateMiddle

TruncateMiddle shortens each line of value to width terminal cells by replacing its center with an ellipsis.
Active SGR styles and OSC 8 hyperlinks are kept with the visible text on either side of the ellipsis.
Values less than one produce an empty string.

```go
fmt.Println(console.TruncateMiddle("abcdefghij", 7))
// abc…hij
```

#### <a id="visiblewidth"></a>VisibleWidth

VisibleWidth returns the largest terminal-cell width among value's lines.
ANSI escapes and combining characters occupy no cells, tabs advance to an eight-cell stop,
and common East Asian and emoji runes occupy two cells.

```go
fmt.Println(console.VisibleWidth("Go界"))
// 4
```

#### <a id="wrap"></a>Wrap

Wrap inserts newlines so each resulting line fits within width terminal cells where possible.
Existing line breaks and ANSI styling are preserved; active SGR styles and OSC 8 hyperlinks are balanced
at each line boundary so they cannot bleed into surrounding layout. Long unbroken words wrap at cell boundaries.
Breakable whitespace at the beginning or end of a resulting line is removed.
Values less than one are returned unchanged.

```go
fmt.Printf("%q\n", console.Wrap("ship the release", 8))
// "ship the\nrelease"
```


### Trees

#### <a id="node"></a>Node

Node creates a tree node and preserves the supplied child order.

```go
console.Tree(console.Node("cmd", console.Node("deploy")))
// cmd
// └── deploy
```

#### <a id="rendertree"></a>RenderTree

RenderTree renders a static tree through the default console.

```go
fmt.Println(console.RenderTree(console.Node("services",
	console.Node("api"),
	console.Node("worker"),
)))
// services
// ├── api
// └── worker
```

#### <a id="tree"></a>Tree

Tree prints a static tree through the default console.

```go
console.Tree(console.Node("project",
	console.Node("cmd", console.Node("deploy")),
	console.Node("README.md"),
))
// project
// ├── cmd
// │   └── deploy
// └── README.md
```

#### <a id="treenode"></a>TreeNode

TreeNode contains one label and its ordered child nodes.

```go
tree := console.TreeNode{
	Label: "project",
	Children: []console.TreeNode{
		{Label: "cmd"},
		{Label: "README.md"},
	},
}
console.Tree(tree)
// project
// ├── cmd
// └── README.md
```

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

### ASCII borders and centered columns

```go
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
```

### Redirect-safe loader outcomes

```go
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
```

### Determinate progress

```go
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

### Recipe: a deployment lifecycle

```go
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
```

### Recipe: an actionable validation report

```go
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
```

### Recipe: machine stdout and status stderr

```go
var machineOutput bytes.Buffer
var statusOutput bytes.Buffer
color := false
unicode := false
console.SetDefault(console.New(console.Config{
	Stdout:         &machineOutput,
	Stderr:         &statusOutput,
	ColorEnabled:   &color,
	UnicodeEnabled: &unicode,
}))

fmt.Fprintln(console.StdoutWriter(), `{"artifact":"app.tar.gz","status":"ready"}`)
fmt.Fprintln(console.StderrWriter(), "status: uploading app.tar.gz")
fmt.Printf("stdout: %q\n", machineOutput.String())
// stdout: "{\"artifact\":\"app.tar.gz\",\"status\":\"ready\"}\n"
fmt.Printf("stderr: %q\n", statusOutput.String())
// stderr: "status: uploading app.tar.gz\n"
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

The docs and examples are separate Go modules so release archives contain only the library. The README API index uses a generator-owned grouping manifest, while each local API target and code sample is generated from the declaration's GoDoc `Example:` block. Focused workflow examples come from standard Go example tests that execute and verify their inline output. Generation validates its marker pair and every example target before writing, so malformed documentation fails without partially changing the README.

## Documentation

- [API index and examples](#api-index)
- [Complete package documentation](https://pkg.go.dev/github.com/goforj/console)
- [Report a bug or request a feature](https://github.com/goforj/console/issues)

## Releasing

Before tagging a release:

- Run the root, docs, and examples tests and vet commands above, including the race suite.
- Run `go generate .` and confirm the working tree has no generated or module-file diff.
- Choose the next semantic version and review the public API and README output one final time.
- Create an annotated tag with `git tag -a vX.Y.Z -m "vX.Y.Z"`, then push it with `git push origin vX.Y.Z`.

The tag runs the same test matrix before the Go module proxy discovers the release.

## License

Released under the [MIT License](LICENSE).
