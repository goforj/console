package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// sourceExampleCase contains one independently executable API target and its asserted output.
type sourceExampleCase struct {
	identity string
	function string
	code     []apiExample
	expected string
}

// sourceExampleStandardImports maps package qualifiers supported by source-comment examples.
var sourceExampleStandardImports = map[string]string{
	"bytes":   "bytes",
	"errors":  "errors",
	"fmt":     "fmt",
	"strings": "strings",
}

// TestSourceGoDocExamples compiles every rendered source-comment example and verifies its adjacent output.
func TestSourceGoDocExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("source-comment examples require building a temporary command")
	}

	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := parseAPISymbols(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planAPIExamples(symbols)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := sourceExampleCases(symbols, plan)
	if err != nil {
		t.Fatal(err)
	}

	source, err := renderSourceExampleCommand(cases)
	if err != nil {
		t.Fatal(err)
	}
	temporaryDirectory := t.TempDir()
	sourcePath := filepath.Join(temporaryDirectory, "main.go")
	if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
		t.Fatalf("write source example command: %v", err)
	}

	executable := filepath.Join(temporaryDirectory, "source-examples")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	build := exec.Command("go", "build", "-o", executable, sourcePath)
	build.Dir = root
	build.Env = sourceExampleEnvironment(os.Environ())
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build source example command: %v\n%s", err, output)
	}

	for _, example := range cases {
		example := example
		t.Run(example.identity, func(t *testing.T) {
			command := exec.Command(executable, example.identity)
			command.Env = sourceExampleEnvironment(os.Environ())
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("execute source example: %v\n%s", err, output)
			}

			got := normalizeSourceExampleTerminalNewline(string(output))
			want := normalizeSourceExampleTerminalNewline(example.expected)
			if got != want {
				t.Fatalf("source example output = %q, want %q\nactual:\n%s\nexpected:\n%s", got, want, got, want)
			}
		})
	}
}

// sourceExampleCases converts each unique direct README target into one isolated executable case.
func sourceExampleCases(symbols []apiSymbol, plan apiExamplePlan) ([]sourceExampleCase, error) {
	directTargets := make(map[string]struct{}, len(plan.targets))
	for _, target := range plan.targets {
		directTargets[target.identity()] = struct{}{}
	}
	for _, symbol := range symbols {
		if len(symbol.examples) == 0 {
			continue
		}
		if _, ok := directTargets[symbol.identity()]; !ok {
			return nil, fmt.Errorf(
				"%s has a source example but is not a direct README example target",
				symbol.identity(),
			)
		}
	}

	cases := make([]sourceExampleCase, 0, len(plan.targets))
	for index, target := range plan.targets {
		var expected strings.Builder
		for _, example := range target.examples {
			if strings.TrimSpace(example.code) == "" {
				continue
			}
			output, err := sourceExampleOutput(example.code)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", target.identity(), err)
			}
			expected.WriteString(output)
		}
		cases = append(cases, sourceExampleCase{
			identity: target.identity(),
			function: fmt.Sprintf("sourceExample%d", index),
			code:     target.examples,
			expected: expected.String(),
		})
	}
	return cases, nil
}

// renderSourceExampleCommand builds one selector-driven command so the test pays compilation cost once.
func renderSourceExampleCommand(cases []sourceExampleCase) ([]byte, error) {
	imports, err := inferSourceExampleImports(cases)
	if err != nil {
		return nil, err
	}

	var source strings.Builder
	source.WriteString("package main\n\nimport (\n\t\"os\"\n")
	for _, path := range imports {
		fmt.Fprintf(&source, "\t%q\n", path)
	}
	source.WriteString("\t\"github.com/goforj/console\"\n)\n\n")
	source.WriteString("// configureSourceExampleConsole installs deterministic presentation and one merged output stream.\n")
	source.WriteString("func configureSourceExampleConsole() {\n")
	source.WriteString("\tcolor := false\n\tunicode := true\n\tanimations := false\n")
	source.WriteString("\tconsole.SetDefault(console.New(console.Config{\n")
	source.WriteString("\t\tStdout: os.Stdout,\n\t\tStderr: os.Stdout,\n")
	source.WriteString("\t\tColorEnabled: &color,\n\t\tUnicodeEnabled: &unicode,\n")
	source.WriteString("\t\tAnimationsEnabled: &animations,\n\t}))\n}\n\n")

	for _, example := range cases {
		fmt.Fprintf(&source, "// %s executes the source-comment examples for %s.\n", example.function, example.identity)
		fmt.Fprintf(&source, "func %s() {\n\tconfigureSourceExampleConsole()\n", example.function)
		for _, block := range example.code {
			if strings.TrimSpace(block.code) == "" {
				continue
			}
			source.WriteString("\tfunc() {\n")
			for _, line := range strings.Split(block.code, "\n") {
				source.WriteString("\t\t" + line + "\n")
			}
			source.WriteString("\t}()\n")
		}
		source.WriteString("}\n\n")
	}

	source.WriteString("// main selects one source-comment target so package global state starts clean.\n")
	source.WriteString("func main() {\n\tif len(os.Args) != 2 {\n\t\tpanic(\"expected one source example selector\")\n\t}\n")
	source.WriteString("\tswitch os.Args[1] {\n")
	for _, example := range cases {
		fmt.Fprintf(&source, "\tcase %s:\n\t\t%s()\n", strconv.Quote(example.identity), example.function)
	}
	source.WriteString("\tdefault:\n\t\tpanic(\"unknown source example selector\")\n\t}\n}\n")

	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format source example command: %w\n%s", err, source.String())
	}
	return formatted, nil
}

// inferSourceExampleImports finds standard-library package qualifiers used by the annotated snippets.
func inferSourceExampleImports(cases []sourceExampleCase) ([]string, error) {
	paths := make(map[string]struct{})
	fileSet := token.NewFileSet()
	for _, example := range cases {
		for _, block := range example.code {
			wrapped := "package main\nfunc example() {\n" + block.code + "\n}\n"
			file, err := parser.ParseFile(fileSet, example.identity+".go", wrapped, 0)
			if err != nil {
				return nil, fmt.Errorf("parse %s source example: %w", example.identity, err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				qualifier, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if path, ok := sourceExampleStandardImports[qualifier.Name]; ok {
					paths[path] = struct{}{}
				}
				return true
			})
		}
	}

	imports := make([]string, 0, len(paths))
	for path := range paths {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	return imports, nil
}

// sourceExampleOutput extracts plain output rows while retaining internal and trailing blank rows.
func sourceExampleOutput(code string) (string, error) {
	var rows []string
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "//":
			rows = append(rows, "")
		case strings.HasPrefix(line, "// ") && !strings.HasPrefix(line, "// #"):
			rows = append(rows, strings.TrimPrefix(line, "// "))
		}
	}
	if len(rows) == 0 {
		return "", errors.New("source example has no plain // output comment")
	}
	return strings.Join(rows, "\n") + "\n", nil
}

// normalizeSourceExampleTerminalNewline ignores only the optional final line terminator.
func normalizeSourceExampleTerminalNewline(output string) string {
	output = strings.TrimSuffix(output, "\n")
	return strings.TrimSuffix(output, "\r")
}

// sourceExampleEnvironment supplies stable UTF-8 terminal capabilities for presentation assertions.
func sourceExampleEnvironment(environment []string) []string {
	return append(environment, "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "TERM=xterm-256color")
}
