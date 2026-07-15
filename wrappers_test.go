package console

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// TestConsoleMethodsHavePackageHelpers keeps the two invocation styles structurally equivalent.
func TestConsoleMethodsHavePackageHelpers(t *testing.T) {
	t.Parallel()

	helpers := map[string]any{
		"Action":          Action,
		"ActionMark":      ActionMark,
		"Actionf":         Actionf,
		"Ask":             Ask,
		"AskDefault":      AskDefault,
		"Box":             Box,
		"Choose":          Choose,
		"Colorize":        Colorize,
		"Confirm":         Confirm,
		"Debug":           Debug,
		"DebugMark":       DebugMark,
		"Debugf":          Debugf,
		"Error":           Error,
		"ErrorMark":       ErrorMark,
		"Errorf":          Errorf,
		"ExpandTabs":      ExpandTabs,
		"Fatal":           Fatal,
		"Fatalf":          Fatalf,
		"Indent":          Indent,
		"Info":            Info,
		"InfoMark":        InfoMark,
		"Infof":           Infof,
		"IsInteractive":   IsInteractive,
		"KeyValueMap":     KeyValueMap,
		"KeyValues":       KeyValues,
		"List":            List,
		"Loader":          NewLoader,
		"NewLine":         NewLine,
		"NumberedList":    NumberedList,
		"PadRight":        PadRight,
		"Print":           Print,
		"Printf":          Printf,
		"Println":         Println,
		"RenderBox":       RenderBox,
		"RenderTable":     RenderTable,
		"Rule":            Rule,
		"Section":         Section,
		"StripANSI":       StripANSI,
		"Style":           Style,
		"Success":         Success,
		"SuccessMark":     SuccessMark,
		"Successf":        Successf,
		"SupportsColor":   SupportsColor,
		"SupportsUnicode": SupportsUnicode,
		"Table":           Table,
		"Truncate":        Truncate,
		"VisibleWidth":    VisibleWidth,
		"Warn":            Warn,
		"WarnMark":        WarnMark,
		"Warnf":           Warnf,
		"Width":           Width,
		"Wrap":            Wrap,
	}

	consoleType := reflect.TypeOf((*Console)(nil))
	if got, want := consoleType.NumMethod(), len(helpers); got != want {
		t.Fatalf("Console exported method count = %d, package helper count = %d", got, want)
	}
	for index := 0; index < consoleType.NumMethod(); index++ {
		method := consoleType.Method(index)
		helper, ok := helpers[method.Name]
		if !ok {
			t.Errorf("Console.%s has no package-level helper", method.Name)
			continue
		}
		assertHelperMatchesMethod(t, method, reflect.TypeOf(helper))
	}
}

// assertHelperMatchesMethod verifies a package helper matches a method after its receiver is removed.
func assertHelperMatchesMethod(t *testing.T, method reflect.Method, helper reflect.Type) {
	t.Helper()

	if got, want := helper.IsVariadic(), method.Type.IsVariadic(); got != want {
		t.Errorf("helper for Console.%s variadic = %t, want %t", method.Name, got, want)
	}
	if got, want := helper.NumIn(), method.Type.NumIn()-1; got != want {
		t.Errorf("helper for Console.%s input count = %d, want %d", method.Name, got, want)
		return
	}
	for index := 0; index < helper.NumIn(); index++ {
		if got, want := helper.In(index), method.Type.In(index+1); got != want {
			t.Errorf("helper for Console.%s input %d = %s, want %s", method.Name, index, got, want)
		}
	}
	if got, want := helper.NumOut(), method.Type.NumOut(); got != want {
		t.Errorf("helper for Console.%s output count = %d, want %d", method.Name, got, want)
		return
	}
	for index := 0; index < helper.NumOut(); index++ {
		if got, want := helper.Out(index), method.Type.Out(index); got != want {
			t.Errorf("helper for Console.%s output %d = %s, want %s", method.Name, index, got, want)
		}
	}
}

// TestPackageLayoutHelpersRouteThroughDefault verifies the complete one-shot layout surface uses the configured default.
func TestPackageLayoutHelpersRouteThroughDefault(t *testing.T) {
	previous := Default()
	t.Cleanup(func() {
		SetDefault(previous)
	})

	var output bytes.Buffer
	color := false
	unicode := false
	configured := New(Config{
		Stdout:         &output,
		ColorEnabled:   &color,
		UnicodeEnabled: &unicode,
		Width:          24,
	})
	SetDefault(configured)

	wantBox := configured.RenderBox("ready", BoxTitle("Status"), BoxColor(""))
	if got := RenderBox("ready", BoxTitle("Status"), BoxColor("")); got != wantBox {
		t.Fatalf("RenderBox() = %q, want %q", got, wantBox)
	}
	wantTable := configured.RenderTable([]string{"Name"}, [][]string{{"api"}})
	if got := RenderTable([]string{"Name"}, [][]string{{"api"}}); got != wantTable {
		t.Fatalf("RenderTable() = %q, want %q", got, wantTable)
	}

	Section("Build")
	Rule("State")
	KeyValues(KV("Mode", "test"))
	KeyValueMap(map[string]any{"Port": 8080})
	List("one")
	NumberedList("first")
	Box("ready", BoxTitle("Status"), BoxColor(""))
	Table([]string{"Name"}, [][]string{{"api"}})

	got := output.String()
	for _, fragment := range []string{
		"> Build\n",
		"Mode  test\n",
		"Port  8080\n",
		"- one\n",
		"1. first\n",
		wantBox + "\n",
		wantTable + "\n",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("layout output %q does not contain %q", got, fragment)
		}
	}
}
