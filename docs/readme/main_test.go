package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestParseAPISymbols verifies exhaustive indexing, manifest grouping, and source-comment extraction.
func TestParseAPISymbols(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := `package console

// Console writes semantic output.
//
// Example:
//
//	fmt.Println("console")
//	// console
type Console struct{}

// Infof writes an informational message.
func (c *Console) Infof() {}

// Warnf writes a warning message.
//
// @group This public-doc marker is deliberately ignored.
func (c *Console) Warnf() {}

// New creates a Console.
func New() *Console { return nil }

// Infof writes through the default Console.
//
// Example: formatted value
//
//	Infof("version %d", 2)
//	// version 2
func Infof() {}

// Warnf writes through the default Console.
func Warnf() {}

// Loader reports background activity.
//
// Example:
//
//	fmt.Println("loader")
//	// loader
type Loader struct{}

// Stop finishes a Loader.
func (l *Loader) Stop() {}

// Version reports build information.
func Version() string { return "" }

// ANSI styles are composable presentation constants.
const (
	// ColorRed colors text red.
	//
	// Example:
	//
	//	fmt.Println(ColorRed)
	//	// red
	ColorRed = "red"
	// ColorBlue colors text blue.
	//
	// Example:
	//
	//	fmt.Println(ColorBlue)
	//	// blue
	ColorBlue = "blue"
)

// ErrClosed indicates a closed console.
//
// Example:
//
//	fmt.Println(ErrClosed)
//	// closed
var ErrClosed = errors.New("closed")

// privateHelper stays outside the public index.
func privateHelper() {}

type Undocumented struct{}

type internalRuntime struct{}

// Start is exported only to satisfy an internal interface.
func (i *internalRuntime) Start() {}
`
	if err := os.WriteFile(filepath.Join(root, "console.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := []apiGroupDefinition{
		{name: "Activity", symbols: []string{"Loader"}},
		{name: "Core", symbols: []string{"Console", "Infof", "New"}},
		{name: "Errors", symbols: []string{"ErrClosed"}},
		{name: "Messages", symbols: []string{"Warnf"}},
		{name: "Other", symbols: []string{"Version"}},
		{name: "Styling", symbols: []string{"ColorBlue", "ColorRed"}},
	}
	got, err := parseAPISymbolsWithManifest(root, manifest)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"Loader|Activity",
		"Loader.Stop|Activity",
		"Console|Core",
		"Infof|Core",
		"New|Core",
		"Console.Infof|Core",
		"ErrClosed|Errors",
		"Warnf|Messages",
		"Console.Warnf|Messages",
		"Version|Other",
		"ColorBlue|Styling",
		"ColorRed|Styling",
	}
	identities := make([]string, 0, len(got))
	for _, symbol := range got {
		identities = append(identities, symbol.identity()+"|"+symbol.group)
	}
	if !reflect.DeepEqual(identities, want) {
		t.Fatalf("parseAPISymbols() identities = %#v, want %#v", identities, want)
	}

	var info apiSymbol
	var red apiSymbol
	for _, symbol := range got {
		switch symbol.identity() {
		case "Infof":
			info = symbol
		case "ColorRed":
			red = symbol
		}
	}
	if info.description != "Infof writes through the default Console." {
		t.Fatalf("Infof description = %q", info.description)
	}
	if len(info.examples) != 1 || info.examples[0].label != "formatted value" ||
		info.examples[0].code != "Infof(\"version %d\", 2)\n// version 2" {
		t.Fatalf("Infof examples = %#v", info.examples)
	}
	if red.description != "ColorRed colors text red." || len(red.examples) != 1 ||
		red.examples[0].code != "fmt.Println(ColorRed)\n// red" {
		t.Fatalf("ColorRed documentation = %#v", red)
	}
}

// TestGenDeclsRenderLocalExamples verifies types, constants, and variables link to their own README code.
func TestGenDeclsRenderLocalExamples(t *testing.T) {
	t.Parallel()

	symbols := []apiSymbol{
		{
			name:        "Config",
			group:       "Runtime",
			description: "Config controls a console.",
			examples:    []apiExample{{code: "fmt.Println(Config{}.Width)\n// 0"}},
		},
		{
			name:        "ColorRed",
			group:       "Styling",
			description: "ColorRed contains the red ANSI prefix.",
			examples:    []apiExample{{code: "fmt.Println(ColorRed != \"\")\n// true"}},
		},
		{
			name:        "ErrClosed",
			group:       "Terminal",
			description: "ErrClosed reports a closed terminal.",
			examples:    []apiExample{{code: "fmt.Println(ErrClosed)\n// closed"}},
		},
	}

	plan, err := planAPIExamples(symbols)
	if err != nil {
		t.Fatal(err)
	}
	index := renderAPI(symbols, plan)
	examples := renderAPIExamples(plan)
	for _, want := range []string{
		"[Config](#config)",
		"[ColorRed](#colorred)",
		"[ErrClosed](#errclosed)",
	} {
		if !strings.Contains(index, want) {
			t.Fatalf("renderAPI() missing %q:\n%s", want, index)
		}
	}
	for _, want := range []string{
		`#### <a id="config"></a>Config`,
		`#### <a id="colorred"></a>ColorRed`,
		`#### <a id="errclosed"></a>ErrClosed`,
	} {
		if !strings.Contains(examples, want) {
			t.Fatalf("renderAPIExamples() missing %q:\n%s", want, examples)
		}
	}
}

// TestGenDeclsRequireExamples verifies exhaustive parsing cannot create an unresolved declaration link.
func TestGenDeclsRequireExamples(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := `package console

// Config controls a console.
type Config struct{}
`
	if err := os.WriteFile(filepath.Join(root, "console.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	symbols, err := parseAPISymbolsWithManifest(root, []apiGroupDefinition{
		{name: "Runtime", symbols: []string{"Config"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planAPIExamples(symbols)
	if err == nil || !strings.Contains(err.Error(), "Config, which has no Example: block") {
		t.Fatalf("planAPIExamples() error = %v, want missing Config example", err)
	}
}

// TestAssignAPIGroupsRejectsInvalidManifest keeps grouping complete and unambiguous as the API grows.
func TestAssignAPIGroupsRejectsInvalidManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		symbols  []apiSymbol
		manifest []apiGroupDefinition
		want     string
	}{
		{
			name:    "missing",
			symbols: []apiSymbol{{name: "Thing"}},
			want:    "missing from the API group manifest",
		},
		{
			name:    "stale",
			symbols: []apiSymbol{{name: "Thing"}},
			manifest: []apiGroupDefinition{
				{name: "Core", symbols: []string{"Thing", "Removed"}},
			},
			want: "stale routing symbol Removed",
		},
		{
			name:    "duplicate",
			symbols: []apiSymbol{{name: "Thing"}},
			manifest: []apiGroupDefinition{
				{name: "Core", symbols: []string{"Thing"}},
				{name: "Other", symbols: []string{"Thing"}},
			},
			want: "assigns Thing to both Core and Other",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := assignAPIGroups(test.symbols, test.manifest)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("assignAPIGroups() error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestParseREADMEExamples verifies tagged standard examples become ordered focused snippets with exact output.
func TestParseREADMEExamples(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	var source strings.Builder
	source.WriteString("package sample_test\n\nimport \"fmt\"\n\n")
	for index, section := range readmeExampleSections {
		name := fmt.Sprintf("Workflow%d", index)
		fmt.Fprintf(
			&source,
			"// Example%s demonstrates a documented workflow.\n//\n// @readme %s\nfunc Example%s() {\n\tfmt.Println(%q)\n\t// %s\n\n\t// Output: %s\n}\n\n",
			name,
			section.id,
			name,
			section.id,
			section.id,
			section.id,
		)
	}
	if err := os.WriteFile(filepath.Join(root, "example_test.go"), []byte(source.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := parseREADMEExamples(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(readmeExampleSections) {
		t.Fatalf("parseREADMEExamples() returned %d examples, want %d", len(got), len(readmeExampleSections))
	}
	for index, example := range got {
		section := readmeExampleSections[index]
		if example.id != section.id || example.title != section.title {
			t.Fatalf("parseREADMEExamples()[%d] = %#v, want category %#v", index, example, section)
		}
		if example.output != section.id+"\n" {
			t.Fatalf("parseREADMEExamples()[%d].output = %q, want %q", index, example.output, section.id+"\n")
		}
		if strings.Contains(example.code, "package main") || strings.Contains(example.code, "func main()") ||
			!strings.Contains(example.code, "fmt.Println") {
			t.Fatalf("parseREADMEExamples()[%d].code is not a focused body snippet:\n%s", index, example.code)
		}
		if strings.Contains(example.code, "Output:") {
			t.Fatalf("parseREADMEExamples()[%d].code retained its output assertion:\n%s", index, example.code)
		}
	}
}

// TestExtractREADMEExamplesRequiresExactOutput rejects selections without deterministic ordered output.
func TestExtractREADMEExamplesRequiresExactOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "missing", output: "", want: "must declare non-empty // Output:"},
		{name: "unordered", output: "\n\t// Unordered output:\n\t// value", want: "require exact output"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fileSet := token.NewFileSet()
			inline := "\n\t// value\n"
			if test.name == "missing" {
				inline = ""
			}
			source := fmt.Sprintf(`package sample_test

import "fmt"

// Example demonstrates a documented workflow.
//
// @readme messages
func Example() {
	fmt.Println("value")%s%s
}
`, inline, test.output)
			file, err := parser.ParseFile(fileSet, "example_test.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}

			_, err = extractREADMEExamples(fileSet, file)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("extractREADMEExamples() error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestOrderREADMEExamplesRejectsInvalidSelections protects the curated set from silent omissions and duplication.
func TestOrderREADMEExamplesRejectsInvalidSelections(t *testing.T) {
	t.Parallel()

	complete := make([]readmeExample, 0, len(readmeExampleSections))
	for _, section := range readmeExampleSections {
		complete = append(complete, readmeExample{id: section.id, name: "Example" + section.id})
	}
	tests := []struct {
		name     string
		examples []readmeExample
		want     string
	}{
		{name: "missing", examples: complete[1:], want: `category "messages" has no tagged`},
		{
			name:     "unknown",
			examples: append(append([]readmeExample(nil), complete...), readmeExample{id: "other", name: "ExampleOther"}),
			want:     `unknown @readme category "other"`,
		},
		{
			name:     "duplicate",
			examples: append(append([]readmeExample(nil), complete...), readmeExample{id: "messages", name: "ExampleAgain"}),
			want:     `category "messages" is used by both`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := orderREADMEExamples(test.examples)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("orderREADMEExamples() error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestRenderAPI verifies global-first columns and local links to resolved source-comment examples.
func TestRenderAPI(t *testing.T) {
	t.Parallel()

	symbols := []apiSymbol{
		{name: "Start", group: "Activity", examples: []apiExample{{code: "Start(\"job\")\n// started"}}},
		{name: "Start", receiver: "Console", group: "Activity"},
		{name: "Start", receiver: "Spinner", group: "Activity", examples: []apiExample{{code: "spinner.Start()\n// started"}}},
		{name: "Start", receiver: "Loader", group: "Activity", examples: []apiExample{{code: "loader.Start()\n// started"}}},
	}

	plan, err := planAPIExamples(symbols)
	if err != nil {
		t.Fatal(err)
	}
	got := renderAPI(symbols, plan)
	want := strings.Join([]string{
		"## API index",
		"",
		"Complete declaration documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/goforj/console). The links below open source-comment examples in this README. Package declarations and global helpers come first; `Console` methods provide the isolated equivalent, while loader and progress lifecycle methods remain on their returned values.",
		"",
		"| Group | Package API | Instance and lifecycle API |",
		"| --- | --- | --- |",
		`| Activity | [Start](#start) | [Console.Start](#start) · [Loader.Start](#loader-start) · [Spinner.Start](#spinner-start) |`,
	}, "\n")
	if got != want {
		t.Fatalf("renderAPI() =\n%s\nwant:\n%s", got, want)
	}
}

// TestPlanAPIExamplesInheritsGlobalTargets verifies wrapper methods reuse concise package examples.
func TestPlanAPIExamplesInheritsGlobalTargets(t *testing.T) {
	t.Parallel()

	symbols := []apiSymbol{
		{name: "Info", group: "Messages", examples: []apiExample{{code: "Info(\"ready\")\n// ready"}}},
		{name: "Info", receiver: "Console", group: "Messages"},
		{name: "NewLoader", group: "Loaders", examples: []apiExample{{code: "loader := NewLoader(\"work\")\n// work"}}},
		{name: "Loader", receiver: "Console", group: "Loaders"},
		{name: "NewProgress", group: "Progress", examples: []apiExample{{code: "progress := NewProgress(2, \"work\")\n// work"}}},
		{name: "Progress", receiver: "Console", group: "Progress"},
	}

	plan, err := planAPIExamples(symbols)
	if err != nil {
		t.Fatal(err)
	}
	wantAnchors := map[string]string{
		"Info":             "info",
		"Console.Info":     "info",
		"NewLoader":        "newloader",
		"Console.Loader":   "newloader",
		"NewProgress":      "newprogress",
		"Console.Progress": "newprogress",
	}
	if !reflect.DeepEqual(plan.anchors, wantAnchors) {
		t.Fatalf("planAPIExamples() anchors = %#v, want %#v", plan.anchors, wantAnchors)
	}
	if len(plan.targets) != 3 {
		t.Fatalf("planAPIExamples() rendered %d targets, want 3", len(plan.targets))
	}
}

// TestPlanAPIExamplesRejectsMissingExamples keeps every index link attached to rendered code.
func TestPlanAPIExamplesRejectsMissingExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		symbols []apiSymbol
		want    string
	}{
		{
			name:    "target has no example",
			symbols: []apiSymbol{{name: "Info", group: "Messages"}},
			want:    "has no Example: block",
		},
		{
			name: "mapped target is absent",
			symbols: []apiSymbol{
				{name: "Loader", receiver: "Console", group: "Loaders"},
			},
			want: "missing example target NewLoader",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := planAPIExamples(test.symbols)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("planAPIExamples() error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestPlanAPIExamplesRequiresPlainOutputComments enforces the README's visible call-and-output convention.
func TestPlanAPIExamplesRequiresPlainOutputComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "missing", code: `fmt.Println("ready")`, want: "no plain // output comment"},
		{name: "typed marker", code: "fmt.Println(\"ready\")\n// #string ready", want: "uses a typed // # output marker"},
		{name: "mixed typed marker", code: "fmt.Println(\"ready\")\n// #string ready\n// ready", want: "uses a typed // # output marker"},
		{name: "visible output", code: "fmt.Println(\"ready\")\n// ready"},
		{name: "blank output", code: "fmt.Println()\n//"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := planAPIExamples([]apiSymbol{{
				name:     "Print",
				group:    "Output",
				examples: []apiExample{{code: test.code}},
			}})
			if test.want == "" {
				if err != nil {
					t.Fatalf("planAPIExamples() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("planAPIExamples() error = %v, want text %q", err, test.want)
			}
		})
	}
}

// TestRenderAPIExamplesKeepsReceiverAnchorsDistinct verifies lifecycle methods can share names safely.
func TestRenderAPIExamplesKeepsReceiverAnchorsDistinct(t *testing.T) {
	t.Parallel()

	symbols := []apiSymbol{
		{
			name:        "Start",
			receiver:    "Loader",
			group:       "Activity",
			description: "Start begins loading.",
			examples:    []apiExample{{code: "loader.Start()\n// started"}},
		},
		{
			name:        "Start",
			receiver:    "Progress",
			group:       "Activity",
			description: "Start begins progress.",
			examples:    []apiExample{{code: "progress.Start()\n// started"}},
		},
	}

	plan, err := planAPIExamples(symbols)
	if err != nil {
		t.Fatal(err)
	}
	got := renderAPIExamples(plan)
	for _, want := range []string{
		`#### <a id="loader-start"></a>Loader.Start`,
		`#### <a id="progress-start"></a>Progress.Start`,
		"```go\nloader.Start()\n// started\n```",
		"```go\nprogress.Start()\n// started\n```",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderAPIExamples() missing %q:\n%s", want, got)
		}
	}
}

// TestRenderExamples verifies code and asserted output stay paired in deterministic Markdown sections.
func TestRenderExamples(t *testing.T) {
	t.Parallel()

	examples := []readmeExample{
		{
			title:  "A workflow",
			code:   "println(\"ready\")\n// ready",
			output: "ready\n",
		},
	}
	got := renderExamples(examples)
	want := strings.Join([]string{
		"## Executable examples",
		"",
		"These focused snippets are generated from standard Go example tests. The test suite executes each one and verifies every inline output comment.",
		"",
		"### A workflow",
		"",
		"```go",
		`println("ready")`,
		"// ready",
		"```",
	}, "\n")
	if got != want {
		t.Fatalf("renderExamples() =\n%s\nwant:\n%s", got, want)
	}
}

// TestExtractInlineOutput verifies blank lines and whitespace remain represented in inline output comments.
func TestExtractInlineOutput(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "example.go", `package main

// helper remains outside the extracted output.
func helper() {}

func main() {
	println("first")
	// first
	println("")
	//
	println("  padded")
	//   padded
}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	got, err := extractInlineOutput(fileSet, file)
	if err != nil {
		t.Fatal(err)
	}
	want := "first\n\n  padded\n"
	if got != want {
		t.Fatalf("extractInlineOutput() = %q, want %q", got, want)
	}
}

// TestExtractInlineOutputRejectsDetachedComments keeps examples in the requested call-then-output form.
func TestExtractInlineOutputRejectsDetachedComments(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "example.go", `package main

func main() {
	println("ready")

	// ready
}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	_, err = extractInlineOutput(fileSet, file)
	if err == nil || !strings.Contains(err.Error(), "must immediately follow") {
		t.Fatalf("extractInlineOutput() error = %v, want adjacency error", err)
	}
}

// TestFormatREADMEExampleOmitsSetup verifies deterministic harness code stays out of focused README snippets.
func TestFormatREADMEExampleOmitsSetup(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "example.go", `package main

func main() {
	// @readme:setup:start
	configureForTest()
	// @readme:setup:end
	println("ready")
	// ready
}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	got, err := formatREADMEExample(fileSet, file)
	if err != nil {
		t.Fatal(err)
	}
	want := "println(\"ready\")\n// ready"
	if got != want {
		t.Fatalf("formatREADMEExample() = %q, want %q", got, want)
	}
}

// TestRenderAPIWithoutSymbols keeps a new repository's generated section useful before its first public declaration.
func TestRenderAPIWithoutSymbols(t *testing.T) {
	t.Parallel()

	got := renderAPI(nil, apiExamplePlan{})
	if !strings.Contains(got, "No documented exported API is available yet.") {
		t.Fatalf("renderAPI(nil) = %q", got)
	}
}

// TestReplaceMarkedSection verifies that generation preserves content outside the requested marker pair.
func TestReplaceMarkedSection(t *testing.T) {
	t.Parallel()

	document := "before<!-- start -->old<!-- end -->after"
	got, err := replaceMarkedSection(document, "<!-- start -->", "<!-- end -->", "new", "example")
	if err != nil {
		t.Fatal(err)
	}

	want := "before<!-- start -->new<!-- end -->after"
	if got != want {
		t.Fatalf("replaceMarkedSection() = %q, want %q", got, want)
	}
}

// TestReplaceMarkedSectionRejectsMalformedMarkers protects hand-written README content from ambiguous generation.
func TestReplaceMarkedSectionRejectsMalformedMarkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		document string
		want     string
	}{
		{name: "missing start", document: "<!-- end -->", want: "start marker"},
		{name: "missing end", document: "<!-- start -->", want: "end marker"},
		{name: "repeated start", document: "<!-- start --><!-- start --><!-- end -->", want: "appears 2 times"},
		{name: "repeated end", document: "<!-- start --><!-- end --><!-- end -->", want: "appears 2 times"},
		{name: "reversed", document: "<!-- end --><!-- start -->", want: "end marker precedes start marker"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := replaceMarkedSection(test.document, "<!-- start -->", "<!-- end -->", "new", "example")
			if err == nil {
				t.Fatal("replaceMarkedSection() error = nil, want malformed marker error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("replaceMarkedSection() error = %q, want text %q", err, test.want)
			}
		})
	}
}
