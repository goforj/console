package console

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// newLayoutTestConsole creates a deterministic console and captures its ordinary output.
func newLayoutTestConsole(width int, unicodeEnabled, colorEnabled bool) (*Console, *bytes.Buffer) {
	output := &bytes.Buffer{}
	console := New(Config{
		Stdout:         output,
		Width:          width,
		UnicodeEnabled: &unicodeEnabled,
		ColorEnabled:   &colorEnabled,
	})
	return console, output
}

// TestSectionRendersUnicodeAndASCII verifies section marks follow terminal capabilities.
func TestSectionRendersUnicodeAndASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		unicodeEnabled bool
		want           string
	}{
		{name: "Unicode", unicodeEnabled: true, want: "◇ Build\n"},
		{name: "ASCII", unicodeEnabled: false, want: "> Build\n"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			console, output := newLayoutTestConsole(20, test.unicodeEnabled, false)
			console.Section("Build")
			if got := output.String(); got != test.want {
				t.Fatalf("Section() wrote %q, want %q", got, test.want)
			}
		})
	}
}

// TestSectionAppliesConfiguredStyles verifies semantic heading colors remain independently reset.
func TestSectionAppliesConfiguredStyles(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, true)
	console.Section("Build")
	want := ColorCyan + "◇" + ColorReset + " " + ColorBoldWhite + "Build" + ColorReset + "\n"
	if got := output.String(); got != want {
		t.Fatalf("Section() wrote %q, want %q", got, want)
	}
}

// TestRuleHonorsWidthAndTitle verifies titled, untitled, truncated, ASCII, and narrow rules.
func TestRuleHonorsWidthAndTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		width          int
		unicodeEnabled bool
		title          string
		want           string
	}{
		{name: "untitled", width: 6, unicodeEnabled: true, want: "──────\n"},
		{name: "titled", width: 10, unicodeEnabled: true, title: "Hi", want: "── Hi ────\n"},
		{name: "truncated title", width: 10, unicodeEnabled: true, title: "abcdefghij", want: "── abcde… \n"},
		{name: "ASCII", width: 10, unicodeEnabled: false, title: "Hi", want: "-- Hi ----\n"},
		{name: "ASCII truncated title", width: 10, unicodeEnabled: false, title: "abcdefghij", want: "-- abcde. \n"},
		{name: "title omitted when narrow", width: 4, unicodeEnabled: true, title: "Hi", want: "────\n"},
		{name: "minimum width", width: 1, unicodeEnabled: false, title: "Hi", want: "-\n"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			console, output := newLayoutTestConsole(test.width, test.unicodeEnabled, false)
			console.Rule(test.title)
			if got := output.String(); got != test.want {
				t.Fatalf("Rule(%q) wrote %q, want %q", test.title, got, test.want)
			}
			if got := VisibleWidth(strings.TrimSuffix(output.String(), "\n")); got != test.width {
				t.Fatalf("Rule(%q) width = %d, want %d", test.title, got, test.width)
			}
		})
	}
}

// TestLayoutMetadataIsSingleLine verifies headings and keys cannot inject physical rows into structured output.
func TestLayoutMetadataIsSingleLine(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	console.Section("Build\nNow")
	console.KeyValues(KV("Long\nKey", "value"))
	want := "◇ Build Now\nLong Key  value\n"
	if got := output.String(); got != want {
		t.Fatalf("structured layout wrote %q, want %q", got, want)
	}
}

// TestRuleStylingDoesNotChangeGeometry ensures ANSI styling remains outside width calculations.
func TestRuleStylingDoesNotChangeGeometry(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(10, true, true)
	console.Rule("Hi")
	if got, want := output.String(), ColorGray+"── Hi ────"+ColorReset+"\n"; got != want {
		t.Fatalf("Rule() wrote %q, want %q", got, want)
	}
	if got := VisibleWidth(strings.TrimSuffix(output.String(), "\n")); got != 10 {
		t.Fatalf("styled Rule() width = %d, want 10", got)
	}
}

// TestKVConstructsAnEntry verifies the convenience constructor preserves key and value identity.
func TestKVConstructsAnEntry(t *testing.T) {
	t.Parallel()

	value := []string{"a", "b"}
	entry := KV("letters", value)
	if entry.Key != "letters" {
		t.Fatalf("KV().Key = %q, want letters", entry.Key)
	}
	if got := fmt.Sprint(entry.Value); got != "[a b]" {
		t.Fatalf("KV().Value = %q, want %q", got, "[a b]")
	}
}

// TestKeyValuesAlignsAndWraps verifies visible key alignment and hanging value indentation.
func TestKeyValuesAlignsAndWraps(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(16, true, false)
	console.KeyValues(
		KV("Name", "Ada"),
		KV("界", "abcdefghijk"),
	)
	want := "Name  Ada\n" +
		"界    abcdefghij\n" +
		"      k\n"
	if got := output.String(); got != want {
		t.Fatalf("KeyValues() wrote:\n%q\nwant:\n%q", got, want)
	}
}

// TestKeyValuesTruncatesLongKeys verifies both columns fit when a key exceeds its allocation.
func TestKeyValuesTruncatesLongKeys(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(10, true, false)
	console.KeyValues(KV("abcdefghijk", "abcdef"))
	want := "abcd…  abc\n" +
		"       def\n"
	if got := output.String(); got != want {
		t.Fatalf("KeyValues() wrote %q, want %q", got, want)
	}
	for index, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := VisibleWidth(line); width > 10 {
			t.Fatalf("line %d width = %d, want at most 10: %q", index, width, line)
		}
	}
}

// TestKeyValuesUsesConfiguredNarrowWidth verifies a structurally feasible two-column row does not overflow.
func TestKeyValuesUsesConfiguredNarrowWidth(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(4, true, false)
	console.KeyValues(KV("abcd", "xy"))
	for index, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := VisibleWidth(line); width > 4 {
			t.Fatalf("line %d width = %d, want at most 4: %q", index, width, line)
		}
	}
}

// TestKeyValuesTruncatesWideValuesAtNarrowWidths verifies a two-cell glyph cannot overflow a one-cell value allocation.
func TestKeyValuesTruncatesWideValuesAtNarrowWidths(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(4, true, false)
	console.KeyValues(KV("A", "界"))
	if got, want := output.String(), "A  …\n"; got != want {
		t.Fatalf("KeyValues() wrote %q, want %q", got, want)
	}
}

// TestKeyValuesHandlesANSIAndWideKeys verifies escape bytes and CJK cells align by visible width.
func TestKeyValuesHandlesANSIAndWideKeys(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(12, true, false)
	console.KeyValues(
		KV(ColorRed+"ID"+ColorReset, "ok"),
		KV("界", "yes"),
	)
	if got, want := StripANSI(output.String()), "ID  ok\n界  yes\n"; got != want {
		t.Fatalf("StripANSI(KeyValues()) = %q, want %q", got, want)
	}
	if !strings.Contains(output.String(), ColorRed+"ID"+ColorReset) {
		t.Fatalf("KeyValues() discarded caller styling: %q", output.String())
	}
}

// TestKeyValueMapIsDeterministic verifies map entries are sorted on every rendering.
func TestKeyValueMapIsDeterministic(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	values := map[string]any{
		"z": "last",
		"a": "first",
		"m": "middle",
	}
	want := "a  first\nm  middle\nz  last\n"
	for iteration := 0; iteration < 50; iteration++ {
		output.Reset()
		console.KeyValueMap(values)
		if got := output.String(); got != want {
			t.Fatalf("KeyValueMap() iteration %d wrote %q, want %q", iteration, got, want)
		}
	}
}

// TestListWrapsWithHangingIndentation verifies unordered Unicode and ASCII list geometry.
func TestListWrapsWithHangingIndentation(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(8, true, false)
	console.List("one two three")
	if got, want := output.String(), "• one\n  two\n  three\n"; got != want {
		t.Fatalf("List() wrote %q, want %q", got, want)
	}

	console, output = newLayoutTestConsole(6, true, false)
	console.List("界界界")
	if got, want := output.String(), "• 界界\n  界\n"; got != want {
		t.Fatalf("List(CJK) wrote %q, want %q", got, want)
	}

	console, output = newLayoutTestConsole(8, false, false)
	console.List("alpha")
	if got, want := output.String(), "- alpha\n"; got != want {
		t.Fatalf("ASCII List() wrote %q, want %q", got, want)
	}
}

// TestListsTruncateWideItemsAtStructuralMinimums verifies prefixes remain aligned when only one content cell is available.
func TestListsTruncateWideItemsAtStructuralMinimums(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(3, true, false)
	console.List("界")
	if got, want := output.String(), "• …\n"; got != want {
		t.Fatalf("List() wrote %q, want %q", got, want)
	}

	console, output = newLayoutTestConsole(4, true, false)
	console.NumberedList("界")
	if got, want := output.String(), "1. …\n"; got != want {
		t.Fatalf("NumberedList() wrote %q, want %q", got, want)
	}
}

// TestListPreservesANSIText verifies caller styling does not consume content width.
func TestListPreservesANSIText(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(8, true, false)
	console.List(ColorGreen + "abcdefghi" + ColorReset)
	if got, want := StripANSI(output.String()), "• abcdef\n  ghi\n"; got != want {
		t.Fatalf("StripANSI(List()) = %q, want %q", got, want)
	}
	for index, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := VisibleWidth(line); width > 8 {
			t.Fatalf("line %d width = %d, want at most 8: %q", index, width, line)
		}
	}
}

// TestNumberedListAlignsPrefixes verifies one- and two-digit markers share one content column.
func TestNumberedListAlignsPrefixes(t *testing.T) {
	t.Parallel()

	items := make([]string, 12)
	for index := range items {
		items[index] = "item"
	}
	console, output := newLayoutTestConsole(12, true, false)
	console.NumberedList(items...)
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != 12 {
		t.Fatalf("NumberedList() line count = %d, want 12", len(lines))
	}
	checks := map[int]string{
		0:  " 1. item",
		8:  " 9. item",
		9:  "10. item",
		11: "12. item",
	}
	for index, want := range checks {
		if got := lines[index]; got != want {
			t.Fatalf("NumberedList() line %d = %q, want %q", index, got, want)
		}
	}
}

// TestNumberedListWrapsWithHangingIndentation verifies continuation lines align after numeric prefixes.
func TestNumberedListWrapsWithHangingIndentation(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(9, true, false)
	console.NumberedList("abcdefghij")
	if got, want := output.String(), "1. abcdef\n   ghij\n"; got != want {
		t.Fatalf("NumberedList() wrote %q, want %q", got, want)
	}
}

// TestEmptyLayoutCollectionsWriteNothing verifies no-op helpers do not add blank lines.
func TestEmptyLayoutCollectionsWriteNothing(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	console.KeyValues()
	console.KeyValueMap(nil)
	console.List()
	console.NumberedList()
	if got := output.String(); got != "" {
		t.Fatalf("empty layout helpers wrote %q, want no output", got)
	}
}

// TestLayoutRenderersReturnComposableText verifies rendering has no output side effects or trailing newline.
func TestLayoutRenderersReturnComposableText(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	tests := []struct {
		name   string
		render func() string
		want   string
	}{
		{name: "section", render: func() string { return console.RenderSection("Build") }, want: "◇ Build"},
		{name: "rule", render: func() string { return console.RenderRule("State") }, want: "── State ───────────"},
		{name: "key values", render: func() string { return console.RenderKeyValues(KV("Mode", "test")) }, want: "Mode  test"},
		{
			name: "key value map",
			render: func() string {
				return console.RenderKeyValueMap(map[string]any{"z": "last", "a": "first"})
			},
			want: "a  first\nz  last",
		},
		{name: "list", render: func() string { return console.RenderList("alpha", "beta") }, want: "• alpha\n• beta"},
		{
			name:   "numbered list",
			render: func() string { return console.RenderNumberedList("alpha", "beta") },
			want:   "1. alpha\n2. beta",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if got := test.render(); got != test.want {
				t.Fatalf("renderer returned %q, want %q", got, test.want)
			}
			if strings.HasSuffix(test.render(), "\n") {
				t.Fatalf("renderer returned a trailing newline: %q", test.render())
			}
		})
	}
	if got := output.String(); got != "" {
		t.Fatalf("renderers wrote %q, want no output", got)
	}
}

// TestLayoutPrintersDelegateToRenderers verifies each printer adds one newline to its rendered form.
func TestLayoutPrintersDelegateToRenderers(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	tests := []struct {
		name     string
		rendered string
		print    func()
	}{
		{name: "section", rendered: console.RenderSection("Build"), print: func() { console.Section("Build") }},
		{name: "rule", rendered: console.RenderRule("State"), print: func() { console.Rule("State") }},
		{
			name:     "key values",
			rendered: console.RenderKeyValues(KV("Mode", "test")),
			print:    func() { console.KeyValues(KV("Mode", "test")) },
		},
		{
			name:     "key value map",
			rendered: console.RenderKeyValueMap(map[string]any{"Mode": "test"}),
			print:    func() { console.KeyValueMap(map[string]any{"Mode": "test"}) },
		},
		{name: "list", rendered: console.RenderList("alpha"), print: func() { console.List("alpha") }},
		{
			name:     "numbered list",
			rendered: console.RenderNumberedList("alpha"),
			print:    func() { console.NumberedList("alpha") },
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			output.Reset()
			test.print()
			if got, want := output.String(), test.rendered+"\n"; got != want {
				t.Fatalf("printer wrote %q, want %q", got, want)
			}
		})
	}
}

// TestEmptyLayoutRenderersStayEmpty verifies composition helpers do not manufacture blank rows.
func TestEmptyLayoutRenderersStayEmpty(t *testing.T) {
	t.Parallel()

	console, _ := newLayoutTestConsole(20, true, false)
	tests := map[string]string{
		"key values":    console.RenderKeyValues(),
		"key value map": console.RenderKeyValueMap(nil),
		"list":          console.RenderList(),
		"numbered list": console.RenderNumberedList(),
	}
	for name, got := range tests {
		if got != "" {
			t.Errorf("%s renderer returned %q, want empty", name, got)
		}
	}
}

// TestPackageRenderHelpersUseDefaultPresentation verifies composable globals honor the active console without writing.
func TestPackageRenderHelpersUseDefaultPresentation(t *testing.T) {
	previous := Default()
	t.Cleanup(func() { SetDefault(previous) })

	configured, output := newLayoutTestConsole(12, false, false)
	SetDefault(configured)
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "section", got: RenderSection("Build"), want: "> Build"},
		{name: "rule", got: RenderRule("State"), want: "-- State ---"},
		{name: "key values", got: RenderKeyValues(KV("Mode", "test")), want: "Mode  test"},
		{name: "key value map", got: RenderKeyValueMap(map[string]any{"B": 2, "A": 1}), want: "A  1\nB  2"},
		{name: "list", got: RenderList("alpha"), want: "- alpha"},
		{name: "numbered list", got: RenderNumberedList("alpha"), want: "1. alpha"},
		{name: "tree", got: RenderTree(Node("root", Node("child"))), want: "root\n`-- child"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s helper = %q, want %q", test.name, test.got, test.want)
		}
	}
	if got := output.String(); got != "" {
		t.Fatalf("render helpers wrote %q, want no output", got)
	}
	Tree(Node("root", Node("child")))
	if got, want := output.String(), "root\n`-- child\n"; got != want {
		t.Fatalf("Tree() wrote %q, want %q", got, want)
	}
}
