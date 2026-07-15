// Command readme rebuilds the generated API index and executable examples in README.md.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	apiStart      = "<!-- api:embed:start -->"
	apiEnd        = "<!-- api:embed:end -->"
	documentation = "https://pkg.go.dev/github.com/goforj/console"
	setupStart    = "// @readme:setup:start"
	setupEnd      = "// @readme:setup:end"
)

var (
	groupHeader         = regexp.MustCompile(`(?im)^\s*@group\s+(.+?)\s*$`)
	readmeExampleHeader = regexp.MustCompile(`(?im)^\s*@readme\s+([a-z][a-z0-9-]*)\s*$`)
)

// readmeExampleSections defines the deliberately focused set of workflows represented in the README.
var readmeExampleSections = []struct {
	id    string
	title string
}{
	{id: "messages", title: "Semantic and multiline messages"},
	{id: "output", title: "Plain output and coordinated writers"},
	{id: "styling", title: "Adaptive styles and marks"},
	{id: "summaries", title: "Sections, rules, and summaries"},
	{id: "lists", title: "Bulleted and numbered lists"},
	{id: "trees", title: "Trees"},
	{id: "boxes", title: "Boxes"},
	{id: "tables", title: "Tables"},
	{id: "table-options", title: "Compact, fixed, and aligned tables"},
	{id: "loader", title: "Redirect-safe loader outcomes"},
	{id: "progress", title: "Determinate progress"},
	{id: "prompts", title: "Questions, defaults, and confirmation"},
	{id: "selection", title: "Choices and secret input"},
	{id: "text", title: "ANSI-aware text utilities"},
	{id: "instance", title: "Isolated console instances"},
}

// apiSymbol contains the public identity and grouping needed for the compact README index.
type apiSymbol struct {
	name     string
	receiver string
	group    string
}

// readmeExample contains one tested standard Go example prepared for Markdown rendering.
type readmeExample struct {
	id     string
	title  string
	name   string
	code   string
	output string
}

// main keeps routine documentation failures concise for local and CI use.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "readme generator:", err)
		os.Exit(1)
	}

	fmt.Println("README.md API index and examples updated")
}

// run computes and validates the complete replacement before it changes README.md.
func run() error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	symbols, err := parseAPISymbols(root)
	if err != nil {
		return fmt.Errorf("parse API declarations: %w", err)
	}
	examples, err := parseREADMEExamples(root)
	if err != nil {
		return fmt.Errorf("parse README examples: %w", err)
	}

	readmePath := filepath.Join(root, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("read README.md: %w", err)
	}

	updated, err := replaceMarkedSection(
		string(readme),
		apiStart,
		apiEnd,
		"\n\n"+renderDocumentation(symbols, examples)+"\n",
		"generated documentation",
	)
	if err != nil {
		return err
	}
	if bytes.Equal(readme, []byte(updated)) {
		return nil
	}

	if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write README.md: %w", err)
	}

	return nil
}

// parseREADMEExamples extracts curated standard Go examples and requires every documented workflow exactly once.
func parseREADMEExamples(root string) ([]readmeExample, error) {
	fileSet := token.NewFileSet()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read repository root: %w", err)
	}

	var examples []readmeExample
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(root, entry.Name())
		file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		extracted, err := extractREADMEExamples(fileSet, file)
		if err != nil {
			return nil, fmt.Errorf("parse %s examples: %w", entry.Name(), err)
		}
		examples = append(examples, extracted...)
	}

	return orderREADMEExamples(examples)
}

// extractREADMEExamples converts tagged standard examples into standalone programs and their asserted output.
func extractREADMEExamples(fileSet *token.FileSet, file *ast.File) ([]readmeExample, error) {
	var examples []readmeExample
	for _, example := range doc.Examples(file) {
		matches := readmeExampleHeader.FindAllStringSubmatch(example.Doc, -1)
		if len(matches) == 0 {
			continue
		}
		name := "Example" + example.Name
		if len(matches) > 1 {
			return nil, fmt.Errorf("%s has %d @readme tags; expected one", name, len(matches))
		}
		if example.Unordered {
			return nil, fmt.Errorf("%s uses unordered output; README examples require exact output", name)
		}
		if example.Output == "" {
			return nil, fmt.Errorf("%s must declare non-empty // Output: text", name)
		}
		if example.Play == nil {
			return nil, fmt.Errorf("%s cannot be converted into a standalone program", name)
		}

		code, err := formatREADMEExample(fileSet, example.Play)
		if err != nil {
			return nil, fmt.Errorf("format %s: %w", name, err)
		}
		inlineOutput, err := extractInlineOutput(example.Play)
		if err != nil {
			return nil, fmt.Errorf("parse %s inline output: %w", name, err)
		}
		if inlineOutput != example.Output {
			return nil, fmt.Errorf(
				"%s inline output comments are %q; want exact // Output: text %q",
				name,
				inlineOutput,
				example.Output,
			)
		}

		examples = append(examples, readmeExample{
			id:     matches[0][1],
			name:   name,
			code:   code,
			output: example.Output,
		})
	}

	return examples, nil
}

// formatREADMEExample renders only the example body so deterministic test wiring does not obscure global helpers.
func formatREADMEExample(fileSet *token.FileSet, file *ast.File) (string, error) {
	var mainFunction *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "main" {
			mainFunction = function
			break
		}
	}
	if mainFunction == nil {
		return "", errors.New("standalone example has no main function")
	}

	comments := make([]*ast.CommentGroup, 0, len(file.Comments))
	for _, group := range file.Comments {
		if group.Pos() >= mainFunction.Body.Pos() && group.End() <= mainFunction.Body.End() {
			comments = append(comments, group)
		}
	}
	snippetFile := &ast.File{
		Name: ast.NewIdent("main"),
		Decls: []ast.Decl{&ast.FuncDecl{
			Name: ast.NewIdent("main"),
			Type: mainFunction.Type,
			Body: mainFunction.Body,
		}},
		Comments: comments,
	}

	var formatted bytes.Buffer
	if err := format.Node(&formatted, fileSet, snippetFile); err != nil {
		return "", err
	}
	value := formatted.String()
	start := strings.Index(value, "func main() {")
	if start < 0 {
		return "", errors.New("formatted example has no main function")
	}
	value = strings.TrimSpace(value[start+len("func main() {"):])
	value = strings.TrimSpace(strings.TrimSuffix(value, "}"))

	lines := strings.Split(value, "\n")
	visible := make([]string, 0, len(lines))
	skippingSetup := false
	foundSetupStart := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case setupStart:
			if skippingSetup || foundSetupStart {
				return "", errors.New("example has duplicate README setup markers")
			}
			foundSetupStart = true
			skippingSetup = true
			continue
		case setupEnd:
			if !skippingSetup {
				return "", errors.New("example has an unmatched README setup end marker")
			}
			skippingSetup = false
			continue
		}
		if skippingSetup {
			continue
		}
		visible = append(visible, strings.TrimPrefix(line, "\t"))
	}
	if skippingSetup {
		return "", errors.New("example has an unmatched README setup start marker")
	}
	return strings.TrimSpace(strings.Join(visible, "\n")), nil
}

// extractInlineOutput collects comments inside main so README output stays next to the call that produced it.
func extractInlineOutput(file *ast.File) (string, error) {
	var body *ast.BlockStmt
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "main" {
			body = function.Body
			break
		}
	}
	if body == nil {
		return "", errors.New("standalone example has no main function")
	}

	var lines []string
	for _, group := range file.Comments {
		if group.Pos() < body.Pos() || group.End() > body.End() {
			continue
		}
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//") {
				return "", errors.New("inline output must use line comments")
			}
			if comment.Text == setupStart || comment.Text == setupEnd {
				continue
			}
			line := strings.TrimPrefix(comment.Text, "//")
			line = strings.TrimPrefix(line, " ")
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return "", errors.New("standalone example has no inline output comments")
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// orderREADMEExamples rejects missing, duplicate, and unknown selections before assigning reader-facing titles.
func orderREADMEExamples(examples []readmeExample) ([]readmeExample, error) {
	selected := make(map[string]readmeExample, len(examples))
	known := make(map[string]struct{}, len(readmeExampleSections))
	for _, section := range readmeExampleSections {
		known[section.id] = struct{}{}
	}
	for _, example := range examples {
		if _, ok := known[example.id]; !ok {
			return nil, fmt.Errorf("%s uses unknown @readme category %q", example.name, example.id)
		}
		if existing, ok := selected[example.id]; ok {
			return nil, fmt.Errorf(
				"README category %q is used by both %s and %s",
				example.id,
				existing.name,
				example.name,
			)
		}
		selected[example.id] = example
	}

	ordered := make([]readmeExample, 0, len(readmeExampleSections))
	for _, section := range readmeExampleSections {
		example, ok := selected[section.id]
		if !ok {
			return nil, fmt.Errorf("README category %q has no tagged standard Go example", section.id)
		}
		example.title = section.title
		ordered = append(ordered, example)
	}
	return ordered, nil
}

// findRoot locates the parent library without relying on a particular source filename.
func findRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for candidate := workingDirectory; ; candidate = filepath.Dir(candidate) {
		if fileExists(filepath.Join(candidate, "go.mod")) && fileExists(filepath.Join(candidate, "README.md")) {
			return filepath.Clean(candidate), nil
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}

	return "", errors.New("could not find library module root")
}

// fileExists treats inaccessible paths as absent because root discovery only probes candidates.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// parseAPISymbols collects documented exported types, functions, and methods from the library package.
func parseAPISymbols(root string) ([]apiSymbol, error) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(
		fileSet,
		root,
		func(info os.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		},
		parser.ParseComments,
	)
	if err != nil {
		return nil, err
	}

	packageName, err := selectPackage(packages)
	if err != nil {
		return nil, err
	}

	pkg, ok := packages[packageName]
	if !ok {
		return nil, fmt.Errorf("selected package %q is missing", packageName)
	}

	typeGroups := collectTypeGroups(pkg)
	var symbols []apiSymbol
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				symbols = append(symbols, documentedTypes(declaration)...)
				symbols = append(symbols, documentedValues(declaration)...)
			case *ast.FuncDecl:
				symbol, include, err := documentedFunction(declaration, typeGroups)
				if err != nil {
					return nil, err
				}
				if include {
					symbols = append(symbols, symbol)
				}
			}
		}
	}

	sortSymbols(symbols)
	return symbols, nil
}

// documentedValues returns public constants and variables while inheriting declaration-level grouping metadata.
func documentedValues(declaration *ast.GenDecl) []apiSymbol {
	if declaration.Tok != token.CONST && declaration.Tok != token.VAR {
		return nil
	}

	declarationGroup, hasDeclarationGroup := explicitGroup(declaration.Doc)
	var symbols []apiSymbol
	for _, specification := range declaration.Specs {
		valueSpecification, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		documentationGroup := valueSpecification.Doc
		if documentationGroup == nil {
			documentationGroup = declaration.Doc
		}
		if documentationGroup == nil {
			continue
		}

		group, hasExplicitGroup := explicitGroup(documentationGroup)
		if !hasExplicitGroup && hasDeclarationGroup {
			group = declarationGroup
		}
		if group == "" {
			group = "Other"
		}
		for _, name := range valueSpecification.Names {
			if ast.IsExported(name.Name) {
				symbols = append(symbols, apiSymbol{name: name.Name, group: group})
			}
		}
	}
	return symbols
}

// selectPackage prefers the largest non-main package so incidental root tools cannot displace the library.
func selectPackage(packages map[string]*ast.Package) (string, error) {
	if len(packages) == 0 {
		return "", errors.New("no packages found in repository root")
	}

	type candidate struct {
		name      string
		fileCount int
	}

	candidates := make([]candidate, 0, len(packages))
	for name, pkg := range packages {
		candidates = append(candidates, candidate{name: name, fileCount: len(pkg.Files)})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].fileCount != candidates[j].fileCount {
			return candidates[i].fileCount > candidates[j].fileCount
		}
		return candidates[i].name < candidates[j].name
	})

	for _, candidate := range candidates {
		if candidate.name != "main" {
			return candidate.name, nil
		}
	}

	return candidates[0].name, nil
}

// collectTypeGroups lets methods inherit their receiver's category without repeating metadata on every method.
func collectTypeGroups(pkg *ast.Package) map[string]string {
	groups := make(map[string]string)
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}

			for _, specification := range general.Specs {
				typeSpecification, ok := specification.(*ast.TypeSpec)
				if !ok || !ast.IsExported(typeSpecification.Name.Name) {
					continue
				}

				documentationGroup := typeSpecification.Doc
				if documentationGroup == nil {
					documentationGroup = general.Doc
				}
				if documentationGroup == nil {
					continue
				}

				groups[typeSpecification.Name.Name] = extractGroup(documentationGroup)
			}
		}
	}

	return groups
}

// documentedTypes returns only public types with GoDoc so the index never advertises undocumented internals.
func documentedTypes(declaration *ast.GenDecl) []apiSymbol {
	if declaration.Tok != token.TYPE {
		return nil
	}

	var symbols []apiSymbol
	for _, specification := range declaration.Specs {
		typeSpecification, ok := specification.(*ast.TypeSpec)
		if !ok || !ast.IsExported(typeSpecification.Name.Name) {
			continue
		}

		documentationGroup := typeSpecification.Doc
		if documentationGroup == nil {
			documentationGroup = declaration.Doc
		}
		if documentationGroup == nil {
			continue
		}

		symbols = append(symbols, apiSymbol{
			name:  typeSpecification.Name.Name,
			group: extractGroup(documentationGroup),
		})
	}

	return symbols
}

// documentedFunction resolves method inheritance while keeping an explicit method group authoritative.
func documentedFunction(function *ast.FuncDecl, typeGroups map[string]string) (apiSymbol, bool, error) {
	if function.Doc == nil || !ast.IsExported(function.Name.Name) {
		return apiSymbol{}, false, nil
	}

	receiver, err := receiverName(function)
	if err != nil {
		return apiSymbol{}, false, fmt.Errorf("%s: %w", function.Name.Name, err)
	}
	if receiver != "" && !ast.IsExported(receiver) {
		return apiSymbol{}, false, nil
	}

	group, hasExplicitGroup := explicitGroup(function.Doc)
	if !hasExplicitGroup && receiver != "" {
		group = typeGroups[receiver]
	}
	if group == "" {
		group = "Other"
	}

	return apiSymbol{name: function.Name.Name, receiver: receiver, group: group}, true, nil
}

// explicitGroup distinguishes absent metadata from an intentional group so receiver inheritance remains possible.
func explicitGroup(documentationGroup *ast.CommentGroup) (string, bool) {
	if documentationGroup == nil {
		return "", false
	}
	match := groupHeader.FindStringSubmatch(documentationGroup.Text())
	if match == nil {
		return "", false
	}

	group := strings.TrimSpace(match[1])
	return group, group != ""
}

// extractGroup keeps unclassified declarations visible instead of silently dropping new API.
func extractGroup(documentationGroup *ast.CommentGroup) string {
	group, ok := explicitGroup(documentationGroup)
	if !ok {
		return "Other"
	}
	return group
}

// receiverName returns the declared receiver type because both README and pkg.go.dev anchors use it.
func receiverName(function *ast.FuncDecl) (string, error) {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return "", nil
	}

	name := receiverTypeName(function.Recv.List[0].Type)
	if name == "" {
		return "", errors.New("unsupported receiver type in exported method")
	}

	return name, nil
}

// receiverTypeName unwraps pointers and generic syntax without inventing aliases for declared types.
func receiverTypeName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return receiverTypeName(expression.X)
	case *ast.IndexExpr:
		return receiverTypeName(expression.X)
	case *ast.IndexListExpr:
		return receiverTypeName(expression.X)
	default:
		return ""
	}
}

// sortSymbols provides stable output even though parser package maps have randomized iteration order.
func sortSymbols(symbols []apiSymbol) {
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].group != symbols[j].group {
			return symbols[i].group < symbols[j].group
		}
		if symbols[i].displayName() != symbols[j].displayName() {
			return symbols[i].displayName() < symbols[j].displayName()
		}
		return symbols[i].packageAnchor() < symbols[j].packageAnchor()
	})
}

// displayName qualifies methods because a growing toolkit can reuse method names across types.
func (symbol apiSymbol) displayName() string {
	if symbol.receiver == "" {
		return symbol.name
	}
	return symbol.receiver + "." + symbol.name
}

// packageAnchor mirrors pkg.go.dev's receiver-qualified declaration fragments.
func (symbol apiSymbol) packageAnchor() string {
	return symbol.displayName()
}

// readmeAnchor uses a URL-friendly receiver prefix so local fragments cannot collide.
func (symbol apiSymbol) readmeAnchor() string {
	if symbol.receiver == "" {
		return strings.ToLower(symbol.name)
	}
	return strings.ToLower(symbol.receiver + "-" + symbol.name)
}

// renderDocumentation keeps the exhaustive API index compact and follows it with selected workflow examples.
func renderDocumentation(symbols []apiSymbol, examples []readmeExample) string {
	return renderAPI(symbols) + "\n\n" + renderExamples(examples)
}

// renderAPI groups a compact index while leaving detailed examples and contracts on pkg.go.dev.
func renderAPI(symbols []apiSymbol) string {
	ordered := append([]apiSymbol(nil), symbols...)
	sortSymbols(ordered)

	var output strings.Builder
	output.WriteString("## API index\n\n")
	output.WriteString("The complete API documentation is available on [pkg.go.dev](")
	output.WriteString(documentation)
	output.WriteString(").\n")

	if len(ordered) == 0 {
		output.WriteString("\nNo documented exported API is available yet.")
		return output.String()
	}

	output.WriteString("\n| Group | API |\n")
	output.WriteString("| --- | --- |\n")
	for start := 0; start < len(ordered); {
		end := start + 1
		for end < len(ordered) && ordered[end].group == ordered[start].group {
			end++
		}

		links := make([]string, 0, end-start)
		for _, symbol := range ordered[start:end] {
			links = append(links, fmt.Sprintf(
				`<a id="%s"></a>[%s](%s#%s)`,
				symbol.readmeAnchor(),
				symbol.displayName(),
				documentation,
				symbol.packageAnchor(),
			))
		}

		fmt.Fprintf(&output, "| %s | %s |\n", ordered[start].group, strings.Join(links, " · "))
		start = end
	}

	return strings.TrimRight(output.String(), "\n")
}

// renderExamples presents each tested snippet with exact output comments beside the calls that produce it.
func renderExamples(examples []readmeExample) string {
	var output strings.Builder
	output.WriteString("## Executable examples\n\n")
	output.WriteString("These focused snippets are generated from standard Go example tests. ")
	output.WriteString("The test suite executes each one and verifies every inline output comment.\n")

	for _, example := range examples {
		output.WriteString("\n### " + example.title + "\n\n")
		output.WriteString("```go\n")
		output.WriteString(example.code)
		output.WriteByte('\n')
		output.WriteString("```\n")
	}

	return strings.TrimRight(output.String(), "\n")
}

// replaceMarkedSection refuses ambiguous marker layouts because guessing could overwrite hand-written documentation.
func replaceMarkedSection(document, startMarker, endMarker, replacement, section string) (string, error) {
	start, end, err := markerBounds(document, startMarker, endMarker, section)
	if err != nil {
		return "", err
	}

	return document[:start] + replacement + document[end:], nil
}

// markerBounds accepts exactly one correctly ordered marker pair and returns only its replaceable interior.
func markerBounds(document, startMarker, endMarker, section string) (int, int, error) {
	startCount := strings.Count(document, startMarker)
	if startCount == 0 {
		return 0, 0, fmt.Errorf("README %s start marker %q is missing", section, startMarker)
	}
	if startCount > 1 {
		return 0, 0, fmt.Errorf(
			"README %s start marker %q appears %d times; expected once",
			section,
			startMarker,
			startCount,
		)
	}

	endCount := strings.Count(document, endMarker)
	if endCount == 0 {
		return 0, 0, fmt.Errorf("README %s end marker %q is missing", section, endMarker)
	}
	if endCount > 1 {
		return 0, 0, fmt.Errorf(
			"README %s end marker %q appears %d times; expected once",
			section,
			endMarker,
			endCount,
		)
	}

	start := strings.Index(document, startMarker) + len(startMarker)
	end := strings.Index(document, endMarker)
	if end < start {
		return 0, 0, fmt.Errorf("README %s markers are malformed: end marker precedes start marker", section)
	}

	return start, end, nil
}
