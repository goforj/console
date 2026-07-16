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
	readmeExampleHeader = regexp.MustCompile(`(?im)^\s*@readme\s+([a-z][a-z0-9-]*)\s*$`)
	apiExampleHeader    = regexp.MustCompile(`(?i)^\s*Example:\s*(.*)$`)
)

// apiGroupDefinition keeps README navigation policy out of public GoDoc.
type apiGroupDefinition struct {
	name    string
	symbols []string
}

// apiGroupManifest assigns every exported package declaration to one reader-facing group.
// Console methods inherit the group of their same-named package declaration, while methods
// on lifecycle types inherit the group of their receiver.
var apiGroupManifest = []apiGroupDefinition{
	{name: "Boxes", symbols: []string{
		"Box", "BoxColor", "BoxOption", "BoxPadding", "BoxTitle", "BoxWidth", "RenderBox",
	}},
	{name: "Layout", symbols: []string{
		"KV", "KeyValue", "KeyValueMap", "KeyValues", "List", "NumberedList",
		"RenderKeyValueMap", "RenderKeyValues", "RenderList", "RenderNumberedList",
		"RenderRule", "RenderSection", "Rule", "Section",
	}},
	{name: "Loaders", symbols: []string{"Loader", "NewLoader"}},
	{name: "Marks", symbols: []string{
		"ActionMark", "DebugMark", "ErrorMark", "InfoMark", "SuccessMark", "WarnMark",
	}},
	{name: "Messages", symbols: []string{
		"Action", "Actionf", "Debug", "Debugf", "Error", "Errorf", "Fatal", "Fatalf",
		"Info", "Infof", "Success", "Successf", "Warn", "Warnf",
	}},
	{name: "Output", symbols: []string{
		"NewLine", "Print", "Printf", "Println", "StderrWriter", "StdoutWriter",
	}},
	{name: "Progress", symbols: []string{"NewProgress", "Progress"}},
	{name: "Prompts", symbols: []string{
		"Ask", "AskDefault", "AskSecret", "Choose", "ChooseIndex", "Confirm", "ErrNonInteractive",
	}},
	{name: "Runtime", symbols: []string{
		"ASCIIMarks", "Config", "Console", "Default", "DefaultMarks", "Marks", "New", "SetDefault",
	}},
	{name: "Styling", symbols: []string{
		"ColorBlack", "ColorBlue", "ColorBoldGreen", "ColorBoldWhite", "ColorCyan", "ColorGray",
		"ColorGreen", "ColorMagenta", "ColorRed", "ColorReset", "ColorWhite", "ColorYellow",
		"Colorize", "Style", "StyleBold", "StyleDim", "StyleUnderline",
	}},
	{name: "Tables", symbols: []string{
		"RenderTable", "Table", "TableCenterAlign", "TableCompact", "TableOption",
		"TableRightAlign", "TableWidths",
	}},
	{name: "Terminal", symbols: []string{
		"ErrTransientActive", "IsInteractive", "SupportsColor", "SupportsUnicode", "Width",
	}},
	{name: "Text", symbols: []string{
		"ExpandTabs", "Indent", "PadCenter", "PadLeft", "PadRight", "StripANSI", "Truncate",
		"TruncateMiddle", "VisibleWidth", "Wrap",
	}},
	{name: "Trees", symbols: []string{"Node", "RenderTree", "Tree", "TreeNode"}},
}

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
	{id: "table-ascii", title: "ASCII borders and centered columns"},
	{id: "loader", title: "Redirect-safe loader outcomes"},
	{id: "progress", title: "Determinate progress"},
	{id: "prompts", title: "Questions, defaults, and confirmation"},
	{id: "selection", title: "Choices and secret input"},
	{id: "text", title: "ANSI-aware text utilities"},
	{id: "deployment-recipe", title: "Recipe: a deployment lifecycle"},
	{id: "validation-recipe", title: "Recipe: an actionable validation report"},
	{id: "ci-recipe", title: "Recipe: machine stdout and status stderr"},
	{id: "instance", title: "Isolated console instances"},
}

// apiSymbol contains one exported declaration and the GoDoc content rendered in the README.
type apiSymbol struct {
	name        string
	receiver    string
	group       string
	description string
	examples    []apiExample
}

// apiExample contains one source-comment example and its position for deterministic ordering.
type apiExample struct {
	label string
	code  string
	line  int
}

// apiExamplePlan maps every indexed symbol to one rendered example target.
type apiExamplePlan struct {
	anchors map[string]string
	targets []apiSymbol
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

	documentationSection, err := renderDocumentation(symbols, examples)
	if err != nil {
		return fmt.Errorf("render API documentation: %w", err)
	}

	updated, err := replaceMarkedSection(
		string(readme),
		apiStart,
		apiEnd,
		"\n\n"+documentationSection+"\n",
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
		inlineOutput, err := extractInlineOutput(fileSet, example.Play)
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

// extractInlineOutput collects adjacent comments inside main so output stays beside its producing statement.
func extractInlineOutput(fileSet *token.FileSet, file *ast.File) (string, error) {
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
		if len(group.List) == 1 && (group.List[0].Text == setupStart || group.List[0].Text == setupEnd) {
			continue
		}
		if err := requireAdjacentOutputComment(fileSet, body, group); err != nil {
			return "", err
		}
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//") {
				return "", errors.New("inline output must use line comments")
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

// requireAdjacentOutputComment rejects comments detached from the statement whose output they document.
func requireAdjacentOutputComment(fileSet *token.FileSet, body *ast.BlockStmt, group *ast.CommentGroup) error {
	var previous ast.Stmt
	for _, statement := range body.List {
		if statement.End() >= group.Pos() {
			break
		}
		previous = statement
	}
	if previous == nil {
		return errors.New("inline output comment has no preceding statement")
	}

	statementLine := fileSet.Position(previous.End()).Line
	commentLine := fileSet.Position(group.Pos()).Line
	if commentLine != statementLine+1 {
		return fmt.Errorf(
			"inline output comment on line %d must immediately follow its statement ending on line %d",
			commentLine,
			statementLine,
		)
	}
	return nil
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

// parseAPISymbols collects and groups documented exported declarations from the library package.
func parseAPISymbols(root string) ([]apiSymbol, error) {
	return parseAPISymbolsWithManifest(root, apiGroupManifest)
}

// parseAPISymbolsWithManifest applies explicit documentation navigation policy to one package.
func parseAPISymbolsWithManifest(root string, manifest []apiGroupDefinition) ([]apiSymbol, error) {
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

	var symbols []apiSymbol
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				symbols = append(symbols, documentedTypes(fileSet, declaration)...)
				symbols = append(symbols, documentedValues(fileSet, declaration)...)
			case *ast.FuncDecl:
				symbol, include, err := documentedFunction(fileSet, declaration)
				if err != nil {
					return nil, err
				}
				if include {
					symbols = append(symbols, symbol)
				}
			}
		}
	}
	if err := assignAPIGroups(symbols, manifest); err != nil {
		return nil, err
	}

	sortSymbols(symbols)
	return symbols, nil
}

// documentedTypes returns public types with the description and examples attached to their declaration.
func documentedTypes(fileSet *token.FileSet, declaration *ast.GenDecl) []apiSymbol {
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

		symbols = append(symbols, documentedPackageSymbol(fileSet, typeSpecification.Name.Name, documentationGroup))
	}
	return symbols
}

// documentedValues returns public constants and variables with their declaration-level GoDoc examples.
func documentedValues(fileSet *token.FileSet, declaration *ast.GenDecl) []apiSymbol {
	if declaration.Tok != token.CONST && declaration.Tok != token.VAR {
		return nil
	}

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

		for _, name := range valueSpecification.Names {
			if ast.IsExported(name.Name) {
				symbols = append(symbols, documentedPackageSymbol(fileSet, name.Name, documentationGroup))
			}
		}
	}
	return symbols
}

// documentedPackageSymbol carries shared declaration GoDoc into one independently linked API symbol.
func documentedPackageSymbol(fileSet *token.FileSet, name string, documentationGroup *ast.CommentGroup) apiSymbol {
	return apiSymbol{
		name:        name,
		description: extractAPIDescription(documentationGroup),
		examples:    extractAPIExamples(fileSet, documentationGroup),
	}
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

// documentedFunction returns a public function or method with GoDoc.
func documentedFunction(fileSet *token.FileSet, function *ast.FuncDecl) (apiSymbol, bool, error) {
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

	return apiSymbol{
		name:        function.Name.Name,
		receiver:    receiver,
		description: extractAPIDescription(function.Doc),
		examples:    extractAPIExamples(fileSet, function.Doc),
	}, true, nil
}

// extractAPIDescription returns the reader-facing prose before source-comment examples.
func extractAPIDescription(group *ast.CommentGroup) string {
	var lines []string
	for _, comment := range group.List {
		line := strings.TrimSpace(apiCommentLine(comment.Text))
		if apiExampleHeader.MatchString(line) {
			break
		}
		if len(lines) == 0 && line == "" {
			continue
		}
		lines = append(lines, line)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// extractAPIExamples returns every source-comment example attached to one declaration.
func extractAPIExamples(fileSet *token.FileSet, group *ast.CommentGroup) []apiExample {
	var examples []apiExample
	var collected []string
	var label string
	var line int
	inExample := false

	flush := func() {
		if len(collected) == 0 {
			return
		}
		code := strings.Join(normalizeAPIExampleIndent(collected), "\n")
		examples = append(examples, apiExample{
			label: label,
			code:  strings.Trim(code, "\n"),
			line:  line,
		})
		collected = nil
		label = ""
		inExample = false
	}

	for _, comment := range group.List {
		raw := apiCommentLine(comment.Text)
		if match := apiExampleHeader.FindStringSubmatch(strings.TrimSpace(raw)); match != nil {
			flush()
			inExample = true
			label = strings.TrimSpace(match[1])
			line = fileSet.Position(comment.Slash).Line
			continue
		}
		if inExample {
			collected = append(collected, raw)
		}
	}
	flush()

	sort.Slice(examples, func(i, j int) bool {
		return examples[i].line < examples[j].line
	})
	return examples
}

// apiCommentLine removes line-comment syntax while retaining code indentation.
func apiCommentLine(text string) string {
	line := strings.TrimPrefix(text, "//")
	if strings.HasPrefix(line, " ") {
		line = line[1:]
	}
	return line
}

// normalizeAPIExampleIndent removes shared indentation without changing nested code.
func normalizeAPIExampleIndent(lines []string) []string {
	minimum := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minimum == -1 || indent < minimum {
			minimum = indent
		}
	}
	if minimum <= 0 {
		return lines
	}

	normalized := make([]string, len(lines))
	for index, line := range lines {
		if len(line) >= minimum {
			normalized[index] = line[minimum:]
			continue
		}
		normalized[index] = strings.TrimLeft(line, " \t")
	}
	return normalized
}

// assignAPIGroups validates the manifest and applies its categories to every documented symbol.
func assignAPIGroups(symbols []apiSymbol, manifest []apiGroupDefinition) error {
	groups := make(map[string]string)
	for _, definition := range manifest {
		if strings.TrimSpace(definition.name) == "" {
			return errors.New("API group manifest contains an empty group name")
		}
		for _, name := range definition.symbols {
			if existing, ok := groups[name]; ok {
				return fmt.Errorf("API group manifest assigns %s to both %s and %s", name, existing, definition.name)
			}
			groups[name] = definition.name
		}
	}

	routingKeys := make(map[string]struct{})
	for _, symbol := range symbols {
		if symbol.receiver == "" {
			routingKeys[symbol.name] = struct{}{}
			continue
		}
		if symbol.receiver != "Console" {
			routingKeys[symbol.receiver] = struct{}{}
		}
	}
	for name := range groups {
		if _, ok := routingKeys[name]; !ok {
			return fmt.Errorf("API group manifest contains stale routing symbol %s", name)
		}
	}

	for index := range symbols {
		groupKey := symbols[index].name
		if symbols[index].receiver != "" && symbols[index].receiver != "Console" {
			groupKey = symbols[index].receiver
		}
		group, ok := groups[groupKey]
		if !ok {
			return fmt.Errorf("documented API symbol %s is missing from the API group manifest", symbols[index].displayName())
		}
		symbols[index].group = group
	}

	return nil
}

// receiverName returns the declared receiver type so local README anchors remain collision-safe.
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
		if (symbols[i].receiver == "") != (symbols[j].receiver == "") {
			return symbols[i].receiver == ""
		}
		if symbols[i].displayName() != symbols[j].displayName() {
			return symbols[i].displayName() < symbols[j].displayName()
		}
		return symbols[i].identity() < symbols[j].identity()
	})
}

// displayName qualifies methods because a growing toolkit can reuse method names across types.
func (symbol apiSymbol) displayName() string {
	if symbol.receiver == "" {
		return symbol.name
	}
	return symbol.receiver + "." + symbol.name
}

// identity distinguishes methods from package helpers with the same exported name.
func (symbol apiSymbol) identity() string {
	return symbol.displayName()
}

// readmeAnchor uses a URL-friendly receiver prefix so local fragments cannot collide.
func (symbol apiSymbol) readmeAnchor() string {
	if symbol.receiver == "" {
		return strings.ToLower(symbol.name)
	}
	return strings.ToLower(symbol.receiver + "-" + symbol.name)
}

// planAPIExamples resolves every index entry to exactly one rendered source-comment example.
func planAPIExamples(symbols []apiSymbol) (apiExamplePlan, error) {
	ordered := append([]apiSymbol(nil), symbols...)
	sortSymbols(ordered)

	byIdentity := make(map[string]apiSymbol, len(ordered))
	for _, symbol := range ordered {
		identity := symbol.identity()
		if _, ok := byIdentity[identity]; ok {
			return apiExamplePlan{}, fmt.Errorf("documented API symbol %s appears more than once", identity)
		}
		byIdentity[identity] = symbol
	}

	plan := apiExamplePlan{anchors: make(map[string]string, len(ordered))}
	rendered := make(map[string]struct{}, len(ordered))
	anchorOwners := make(map[string]string, len(ordered))
	for _, symbol := range ordered {
		targetIdentity := apiExampleTargetIdentity(symbol, byIdentity)
		target, ok := byIdentity[targetIdentity]
		if !ok {
			return apiExamplePlan{}, fmt.Errorf(
				"documented API symbol %s resolves to missing example target %s",
				symbol.identity(),
				targetIdentity,
			)
		}
		if !hasAPIExample(target) {
			return apiExamplePlan{}, fmt.Errorf(
				"documented API symbol %s resolves to %s, which has no Example: block",
				symbol.identity(),
				targetIdentity,
			)
		}

		anchor := target.readmeAnchor()
		plan.anchors[symbol.identity()] = anchor
		if _, ok := rendered[targetIdentity]; ok {
			continue
		}
		if err := requireAPIExampleOutput(target); err != nil {
			return apiExamplePlan{}, err
		}
		if owner, ok := anchorOwners[anchor]; ok {
			return apiExamplePlan{}, fmt.Errorf(
				"API example anchor %q is shared by %s and %s",
				anchor,
				owner,
				targetIdentity,
			)
		}
		anchorOwners[anchor] = targetIdentity
		rendered[targetIdentity] = struct{}{}
		plan.targets = append(plan.targets, target)
	}

	return plan, nil
}

// apiExampleTargetIdentity applies the global-first README policy without hiding instance API entries.
func apiExampleTargetIdentity(symbol apiSymbol, symbols map[string]apiSymbol) string {
	if symbol.receiver != "Console" {
		return symbol.identity()
	}

	switch symbol.name {
	case "Loader":
		return "NewLoader"
	case "Progress":
		return "NewProgress"
	}
	if _, ok := symbols[symbol.name]; ok {
		return symbol.name
	}
	return symbol.identity()
}

// hasAPIExample rejects annotations containing only whitespace so every local link reaches useful code.
func hasAPIExample(symbol apiSymbol) bool {
	for _, example := range symbol.examples {
		if strings.TrimSpace(example.code) != "" {
			return true
		}
	}
	return false
}

// requireAPIExampleOutput enforces the call-then-output convention on every rendered block.
func requireAPIExampleOutput(symbol apiSymbol) error {
	for _, example := range symbol.examples {
		if strings.TrimSpace(example.code) == "" {
			continue
		}

		name := symbol.identity()
		if example.label != "" {
			name += " (" + example.label + ")"
		}
		if hasTypedOutputMarker(example.code) {
			return fmt.Errorf("API example %s uses a typed // # output marker; use plain // output", name)
		}
		if hasPlainOutputComment(example.code) {
			continue
		}
		return fmt.Errorf("API example %s has no plain // output comment", name)
	}
	return nil
}

// hasTypedOutputMarker detects the typed annotations used by generators with a different README convention.
func hasTypedOutputMarker(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "// #") {
			return true
		}
	}
	return false
}

// hasPlainOutputComment recognizes visible or blank output rows written as ordinary line comments.
func hasPlainOutputComment(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "//" {
			return true
		}
		if strings.HasPrefix(line, "// ") {
			return true
		}
	}
	return false
}

// renderDocumentation keeps the declaration index compact and follows it with both example collections.
func renderDocumentation(symbols []apiSymbol, examples []readmeExample) (string, error) {
	plan, err := planAPIExamples(symbols)
	if err != nil {
		return "", err
	}

	return renderAPI(symbols, plan) + "\n\n" + renderAPIExamples(plan) + "\n\n" + renderExamples(examples), nil
}

// renderAPI places package declarations before methods so the concise default path is easiest to scan.
func renderAPI(symbols []apiSymbol, plan apiExamplePlan) string {
	ordered := append([]apiSymbol(nil), symbols...)
	sortSymbols(ordered)

	var output strings.Builder
	output.WriteString("## API index\n\n")
	output.WriteString("Complete declaration documentation is available on [pkg.go.dev](")
	output.WriteString(documentation)
	output.WriteString("). The links below open source-comment examples in this README. Package declarations and global helpers come first; `Console` methods provide the isolated equivalent, while loader and progress lifecycle methods remain on their returned values.\n")

	if len(ordered) == 0 {
		output.WriteString("\nNo documented exported API is available yet.")
		return output.String()
	}

	output.WriteString("\n| Group | Package API | Instance and lifecycle API |\n")
	output.WriteString("| --- | --- | --- |\n")
	for start := 0; start < len(ordered); {
		end := start + 1
		for end < len(ordered) && ordered[end].group == ordered[start].group {
			end++
		}

		packageLinks := make([]string, 0, end-start)
		methodLinks := make([]string, 0, end-start)
		for _, symbol := range ordered[start:end] {
			link := fmt.Sprintf("[%s](#%s)", symbol.displayName(), plan.anchors[symbol.identity()])
			if symbol.receiver == "" {
				packageLinks = append(packageLinks, link)
			} else {
				methodLinks = append(methodLinks, link)
			}
		}

		packageAPI := strings.Join(packageLinks, " · ")
		if packageAPI == "" {
			packageAPI = "—"
		}
		methodAPI := strings.Join(methodLinks, " · ")
		if methodAPI == "" {
			methodAPI = "—"
		}
		fmt.Fprintf(&output, "| %s | %s | %s |\n", ordered[start].group, packageAPI, methodAPI)
		start = end
	}

	return strings.TrimRight(output.String(), "\n")
}

// renderAPIExamples presents each unique source-comment target once under a collision-safe local anchor.
func renderAPIExamples(plan apiExamplePlan) string {
	var output strings.Builder
	output.WriteString("## API examples\n\n")
	output.WriteString("These examples are generated from package GoDoc comments. ")
	output.WriteString("Package-level helpers are shown in preference to equivalent `Console` methods.\n")

	if len(plan.targets) == 0 {
		output.WriteString("\nNo documented API examples are available yet.")
		return output.String()
	}

	for start := 0; start < len(plan.targets); {
		end := start + 1
		for end < len(plan.targets) && plan.targets[end].group == plan.targets[start].group {
			end++
		}

		output.WriteString("\n### " + plan.targets[start].group + "\n\n")
		for _, symbol := range plan.targets[start:end] {
			fmt.Fprintf(&output, "#### <a id=\"%s\"></a>%s\n\n", symbol.readmeAnchor(), symbol.displayName())
			if symbol.description != "" {
				output.WriteString(symbol.description + "\n\n")
			}
			for _, example := range symbol.examples {
				if strings.TrimSpace(example.code) == "" {
					continue
				}
				if example.label != "" && len(symbol.examples) > 1 {
					output.WriteString("_Example: " + example.label + "_\n\n")
				}
				output.WriteString("```go\n")
				output.WriteString(strings.TrimSpace(example.code))
				output.WriteString("\n```\n\n")
			}
		}
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
