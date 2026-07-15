package console

import (
	"strings"
	"testing"
)

// TestStripANSI removes complete CSI, OSC, and ESC sequences.
func TestStripANSI(t *testing.T) {
	t.Parallel()

	value := "\x1b[31mred\x1b[0m " +
		"\x1b]0;window title\aok " +
		"\x1b]8;;https://example.test\x1b\\link\x1b]8;;\x1b\\ " +
		"\x1b7saved \x1b(Bcharset"

	if got, want := StripANSI(value), "red ok link saved charset"; got != want {
		t.Fatalf("StripANSI() = %q, want %q", got, want)
	}
}

// TestStripANSIRetainsIncompleteSequences protects malformed input from silent data loss.
func TestStripANSIRetainsIncompleteSequences(t *testing.T) {
	t.Parallel()

	tests := []string{
		"before\x1b",
		"before\x1b ",
		"before\x1b[31",
		"before\x1b]8;;https://example.test",
		"before\x1b]title\nstill\a",
		"before\x1b]title\x1b",
		"before\x1b😀",
	}
	for _, value := range tests {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if got := StripANSI(value); got != value {
				t.Fatalf("StripANSI(%q) = %q, want the malformed input unchanged", value, got)
			}
		})
	}
}

// TestStripANSIPreservesInvalidUTF8 verifies an ANSI utility does not rewrite unrelated malformed bytes.
func TestStripANSIPreservesInvalidUTF8(t *testing.T) {
	t.Parallel()

	value := string([]byte{'a', 0xff, 'b'})
	if got := StripANSI(value); got != value {
		t.Fatalf("StripANSI(%q) = %q, want the original bytes", value, got)
	}
}

// TestVisibleWidthCountsTerminalCells covers styled, wide, combining, emoji, and tabular text.
func TestVisibleWidthCountsTerminalCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "ASCII", value: "plain", want: 5},
		{name: "CSI styling", value: "\x1b[31mred\x1b[0m", want: 3},
		{name: "OSC hyperlink with BEL", value: "\x1b]8;;https://example.test\aDocs\x1b]8;;\a", want: 4},
		{name: "OSC hyperlink with ST", value: "\x1b]8;;https://example.test\x1b\\Docs\x1b]8;;\x1b\\", want: 4},
		{name: "CJK", value: "界a", want: 3},
		{name: "combining mark", value: "e\u0301", want: 1},
		{name: "emoji ZWJ and modifier", value: "👩🏽‍💻", want: 2},
		{name: "text presentation symbols", value: "✔✖⚠☀✈❤", want: 6},
		{name: "emoji variation selectors", value: "✔️✖️⚠️☀️✈️❤️", want: 12},
		{name: "default emoji presentation", value: "✅", want: 2},
		{name: "text variation selector", value: "✅︎", want: 1},
		{name: "regional indicator flag", value: "🇺🇳", want: 2},
		{name: "tab stop", value: "a\tb", want: 9},
		{name: "control characters", value: "a\rb", want: 2},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := VisibleWidth(test.value); got != test.want {
				t.Fatalf("VisibleWidth(%q) = %d, want %d", test.value, got, test.want)
			}
		})
	}
}

// TestVisibleWidthUsesWidestLine verifies that line breaks reset cell positions and tab stops.
func TestVisibleWidthUsesWidestLine(t *testing.T) {
	t.Parallel()

	if got, want := VisibleWidth("ab\n界界\nx\ty"), 9; got != want {
		t.Fatalf("VisibleWidth() = %d, want %d", got, want)
	}
}

// TestTruncateUsesVisibleCells preserves complete glyphs while applying an ellipsis per line.
func TestTruncateUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "unchanged", value: "short", width: 5, want: "short"},
		{name: "ASCII", value: "abcdef", width: 5, want: "abcd…"},
		{name: "ellipsis only", value: "abcdef", width: 1, want: "…"},
		{name: "zero width", value: "abcdef", width: 0, want: ""},
		{name: "negative width", value: "abcdef", width: -1, want: ""},
		{name: "CJK", value: "界界界", width: 5, want: "界界…"},
		{name: "combining", value: "e\u0301clair", width: 4, want: "e\u0301cl…"},
		{name: "emoji", value: "👩🏽‍💻 developer", width: 5, want: "👩🏽‍💻 d…"},
		{name: "each line", value: "abcdef\n界界界", width: 4, want: "abc…\n界…"},
		{name: "CRLF normalization", value: "abcdef\r\nxy", width: 4, want: "abc…\nxy"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := Truncate(test.value, test.width); got != test.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestTruncateResetsSGRStyling ensures an ellipsis cannot inherit a truncated value's style.
func TestTruncateResetsSGRStyling(t *testing.T) {
	t.Parallel()

	value := ColorRed + "abcdef" + ColorReset
	want := ColorRed + "abc" + ColorReset + "…"
	if got := Truncate(value, 4); got != want {
		t.Fatalf("Truncate() = %q, want %q", got, want)
	}
}

// TestTruncatePreservesOSCHyperlinkText verifies OSC metadata does not affect visible truncation.
func TestTruncatePreservesOSCHyperlinkText(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\a"
	close := "\x1b]8;;\a"
	got := Truncate(open+"abcdef"+close, 4)
	if stripped, want := StripANSI(got), "abc…"; stripped != want {
		t.Fatalf("StripANSI(Truncate()) = %q, want %q", stripped, want)
	}
	if width := VisibleWidth(got); width != 4 {
		t.Fatalf("VisibleWidth(Truncate()) = %d, want 4", width)
	}
	closeIndex := strings.Index(got, close)
	ellipsisIndex := strings.Index(got, "…")
	if closeIndex < 0 || ellipsisIndex < 0 || closeIndex > ellipsisIndex {
		t.Fatalf("Truncate() did not close its OSC hyperlink before the ellipsis: %q", got)
	}
}

// TestTruncateMiddleUsesVisibleCells retains useful context from both ends of every physical line.
func TestTruncateMiddleUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "unchanged", value: "short", width: 5, want: "short"},
		{name: "balanced sides", value: "abcdef", width: 5, want: "ab…ef"},
		{name: "prefix receives odd cell", value: "abcdef", width: 4, want: "ab…f"},
		{name: "ellipsis only", value: "abcdef", width: 1, want: "…"},
		{name: "zero width", value: "abcdef", width: 0, want: ""},
		{name: "negative width", value: "abcdef", width: -1, want: ""},
		{name: "CJK", value: "界界界", width: 5, want: "界…界"},
		{name: "combining", value: "e\u0301clair", width: 4, want: "e\u0301c…r"},
		{name: "emoji", value: "👩🏽‍💻 developer", width: 7, want: "👩🏽‍💻 …per"},
		{name: "each line", value: "abcdef\n界界界", width: 4, want: "ab…f\n界…"},
		{name: "CRLF normalization", value: "abcdef\r\nxy", width: 4, want: "ab…f\nxy"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := TruncateMiddle(test.value, test.width); got != test.want {
				t.Fatalf("TruncateMiddle(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestTruncateMiddleBalancesPresentationMetadata keeps the ellipsis outside styles and hyperlinks.
func TestTruncateMiddleBalancesPresentationMetadata(t *testing.T) {
	t.Parallel()

	styled := ColorRed + "abcdef" + ColorReset
	styledWant := ColorRed + "ab" + ColorReset + "…" + ColorRed + "f" + ColorReset
	if got := TruncateMiddle(styled, 4); got != styledWant {
		t.Fatalf("TruncateMiddle(styled) = %q, want %q", got, styledWant)
	}

	open := "\x1b]8;;https://example.test\a"
	close := "\x1b]8;;\a"
	linked := TruncateMiddle(open+"abcdef"+close, 4)
	if got, want := StripANSI(linked), "ab…f"; got != want {
		t.Fatalf("StripANSI(TruncateMiddle(linked)) = %q, want %q", got, want)
	}
	if !strings.Contains(linked, close+"…"+open) {
		t.Fatalf("TruncateMiddle(linked) did not isolate its ellipsis: %q", linked)
	}

	if got, want := TruncateMiddle(ColorRed+"界界", 2), "…"; got != want {
		t.Fatalf("TruncateMiddle(overwide styled text) = %q, want %q", got, want)
	}
}

// TestWrapUsesVisibleCells wraps words and indivisible glyph clusters without counting ANSI bytes.
func TestWrapUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "words", value: "one two three", width: 8, want: "one two\nthree"},
		{name: "word exactly fills line", value: "hello world", width: 5, want: "hello\nworld"},
		{name: "trailing separator after full line", value: "hello ", width: 5, want: "hello"},
		{name: "long word", value: "abcdefgh", width: 3, want: "abc\ndef\ngh"},
		{name: "CJK", value: "界界界", width: 4, want: "界界\n界"},
		{name: "overwide CJK", value: "界界", width: 1, want: "界\n界"},
		{name: "overwide combining", value: "界\u0301界\u0301", width: 1, want: "界\u0301\n界\u0301"},
		{name: "overwide variation selector", value: "✈️✈️", width: 1, want: "✈️\n✈️"},
		{name: "combining", value: "e\u0301e\u0301e\u0301", width: 2, want: "e\u0301e\u0301\ne\u0301"},
		{name: "emoji ZWJ", value: "👩🏽‍💻👩🏽‍💻", width: 2, want: "👩🏽‍💻\n👩🏽‍💻"},
		{name: "overwide emoji ZWJ", value: "👩🏽‍💻👩🏽‍💻", width: 1, want: "👩🏽‍💻\n👩🏽‍💻"},
		{name: "leading and trailing spaces", value: "  alpha  ", width: 10, want: "alpha"},
		{name: "non-breaking space is retained", value: " alpha\u00a0 ", width: 10, want: "alpha\u00a0"},
		{name: "nonpositive width", value: "alpha beta", width: 0, want: "alpha beta"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := Wrap(test.value, test.width); got != test.want {
				t.Fatalf("Wrap(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestWrapPreservesExplicitLinesAndEscapes retains blank lines and complete styling metadata.
func TestWrapPreservesExplicitLinesAndEscapes(t *testing.T) {
	t.Parallel()

	if got, want := Wrap("a\r\n\r\nb", 3), "a\n\nb"; got != want {
		t.Fatalf("Wrap() = %q, want %q", got, want)
	}

	styled := ColorRed + "abcdef" + ColorReset
	if got, want := Wrap(styled, 3), ColorRed+"abc"+ColorReset+"\n"+ColorRed+"def"+ColorReset; got != want {
		t.Fatalf("Wrap(styled) = %q, want %q", got, want)
	}

	styledTrailingSpace := ColorRed + "alpha " + ColorReset
	if got, want := Wrap(styledTrailingSpace, 10), ColorRed+"alpha"+ColorReset; got != want {
		t.Fatalf("Wrap(styled trailing space) = %q, want %q", got, want)
	}
	for index, line := range strings.Split(Wrap(styled, 3), "\n") {
		if got := VisibleWidth(line); got != 3 {
			t.Fatalf("Wrap(styled) line %d width = %d, want 3", index, got)
		}
		if !strings.HasSuffix(line, ColorReset) {
			t.Fatalf("Wrap(styled) line %d does not end with a reset: %q", index, line)
		}
	}

	open := "\x1b]8;;https://example.test\x1b\\"
	closeST := "\x1b]8;;\x1b\\"
	closeBEL := "\x1b]8;;\a"
	linked := Wrap(open+"abcdef"+closeST, 3)
	if got, want := StripANSI(linked), "abc\ndef"; got != want {
		t.Fatalf("StripANSI(Wrap(linked)) = %q, want %q", got, want)
	}
	linkedLines := strings.Split(linked, "\n")
	for index, line := range linkedLines {
		if !strings.HasPrefix(line, open) {
			t.Fatalf("Wrap(linked) line %d does not reopen its OSC hyperlink: %q", index, line)
		}
		if !strings.HasSuffix(line, closeBEL) && !strings.HasSuffix(line, closeST) {
			t.Fatalf("Wrap(linked) line %d does not close its OSC hyperlink: %q", index, line)
		}
	}
	if strings.Count(linked, open) != 2 || strings.Count(linked, closeBEL)+strings.Count(linked, closeST) != 2 {
		t.Fatalf("Wrap(linked) did not create one bounded OSC hyperlink per line: %q", linked)
	}

	for name, color := range map[string]string{
		"indexed black": "\x1b[38;5;0m",
		"RGB black":     "\x1b[38;2;0;0;0m",
	} {
		name, color := name, color
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			value := StyleBold + color + "abcdef" + ColorReset
			want := StyleBold + color + "abc" + ColorReset + "\n" +
				StyleBold + color + "def" + ColorReset
			if got := Wrap(value, 3); got != want {
				t.Fatalf("Wrap(extended color) = %q, want %q", got, want)
			}
		})
	}
}

// TestPadRightUsesVisibleCells aligns plain, styled, wide, combining, tabbed, and multiline values.
func TestPadRightUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "plain", value: "ab", width: 4, want: "ab  "},
		{name: "already wide", value: "abcdef", width: 4, want: "abcdef"},
		{name: "styled", value: ColorRed + "ab" + ColorReset, width: 4, want: ColorRed + "ab" + ColorReset + "  "},
		{name: "CJK", value: "界", width: 4, want: "界  "},
		{name: "combining", value: "e\u0301", width: 3, want: "e\u0301  "},
		{name: "tab", value: "a\t", width: 10, want: "a\t  "},
		{name: "multiline", value: "a\n界", width: 3, want: "a  \n界 "},
		{name: "CRLF normalization", value: "a\r\nb", width: 2, want: "a \nb "},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := PadRight(test.value, test.width); got != test.want {
				t.Fatalf("PadRight(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestPadLeftUsesVisibleCells aligns plain, styled, wide, tabbed, and multiline values from the right.
func TestPadLeftUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "plain", value: "ab", width: 4, want: "  ab"},
		{name: "already wide", value: "abcdef", width: 4, want: "abcdef"},
		{name: "styled", value: ColorRed + "ab" + ColorReset, width: 4, want: "  " + ColorRed + "ab" + ColorReset},
		{name: "CJK", value: "界", width: 4, want: "  界"},
		{name: "tab expansion", value: "a\t", width: 10, want: "  a       "},
		{name: "multiline", value: "a\n界", width: 3, want: "  a\n 界"},
		{name: "CRLF normalization", value: "a\r\nb", width: 2, want: " a\n b"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := PadLeft(test.value, test.width); got != test.want {
				t.Fatalf("PadLeft(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestPadCenterUsesVisibleCells centers plain, styled, wide, tabbed, and multiline values deterministically.
func TestPadCenterUsesVisibleCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "even padding", value: "ab", width: 6, want: "  ab  "},
		{name: "odd padding", value: "ab", width: 5, want: " ab  "},
		{name: "already wide", value: "abcdef", width: 4, want: "abcdef"},
		{name: "styled", value: ColorRed + "ab" + ColorReset, width: 4, want: " " + ColorRed + "ab" + ColorReset + " "},
		{name: "CJK", value: "界", width: 5, want: " 界  "},
		{name: "tab expansion", value: "a\t", width: 11, want: " a       " + "  "},
		{name: "multiline", value: "a\n界", width: 4, want: " a  \n 界 "},
		{name: "CRLF normalization", value: "a\r\nb", width: 3, want: " a \n b "},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := PadCenter(test.value, test.width); got != test.want {
				t.Fatalf("PadCenter(%q, %d) = %q, want %q", test.value, test.width, got, test.want)
			}
		})
	}
}

// TestExpandTabsUsesLineRelativeCellStops verifies ANSI and wide glyphs affect tab expansion correctly.
func TestExpandTabsUsesLineRelativeCellStops(t *testing.T) {
	t.Parallel()

	value := ColorRed + "a" + ColorReset + "\tb\n界\tz"
	want := ColorRed + "a" + ColorReset + "       b\n界      z"
	if got := ExpandTabs(value); got != want {
		t.Fatalf("ExpandTabs() = %q, want %q", got, want)
	}
	if got := ExpandTabs("plain"); got != "plain" {
		t.Fatalf("ExpandTabs() without tabs = %q, want unchanged input", got)
	}
}

// TestSanitizeLayoutTextRetainsOnlyGeometrySafeTerminalSequences verifies embedded content cannot move the cursor or mutate terminal state.
func TestSanitizeLayoutTextRetainsOnlyGeometrySafeTerminalSequences(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\a"
	close := "\x1b]8;;\a"
	value := ColorRed + "red" + ColorReset +
		"\x1b[20C" +
		"\x1b]0;window title\a" +
		"safe" + open + "link" + close + "\a"
	want := ColorRed + "red" + ColorReset + "safe" + open + "link" + close
	if got := sanitizeLayoutText(value, true); got != want {
		t.Fatalf("sanitizeLayoutText() = %q, want %q", got, want)
	}

	if got, want := sanitizeLayoutText("a\r\nb\rc\n", true), "a\nb\nc\n"; got != want {
		t.Fatalf("sanitizeLayoutText(multiline) = %q, want %q", got, want)
	}
	if got, want := sanitizeLayoutText("a\r\nb\rc\n", false), "a b c "; got != want {
		t.Fatalf("sanitizeLayoutText(single line) = %q, want %q", got, want)
	}
}

// TestSingleLineLayoutTextBalancesCallerMetadata verifies labels cannot leak styles or hyperlinks into generated chrome.
func TestSingleLineLayoutTextBalancesCallerMetadata(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\a"
	got := singleLineLayoutText(StyleBold + "Build\n" + open + "docs")
	if stripped, want := StripANSI(got), "Build docs"; stripped != want {
		t.Fatalf("StripANSI(singleLineLayoutText()) = %q, want %q", stripped, want)
	}
	if !strings.HasSuffix(got, ColorReset+"\x1b]8;;\a") {
		t.Fatalf("singleLineLayoutText() did not close caller metadata: %q", got)
	}
}

// TestIndentPrefixesEveryLine verifies hanging, blank, and trailing lines retain their structure.
func TestIndentPrefixesEveryLine(t *testing.T) {
	t.Parallel()

	if got, want := Indent("a\n\nb\n", "> "), "> a\n> \n> b\n> "; got != want {
		t.Fatalf("Indent() = %q, want %q", got, want)
	}
	if got := Indent("", "> "); got != "" {
		t.Fatalf("Indent(empty) = %q, want empty output", got)
	}
}

// TestConsoleTextHelpersMatchPackageFunctions locks the stateless text surface to both invocation styles.
func TestConsoleTextHelpersMatchPackageFunctions(t *testing.T) {
	t.Parallel()

	console := New(Config{})
	styled := ColorRed + "hello world" + ColorReset

	if got, want := console.StripANSI(styled), StripANSI(styled); got != want {
		t.Fatalf("Console.StripANSI() = %q, want %q", got, want)
	}
	if got, want := console.VisibleWidth(styled), VisibleWidth(styled); got != want {
		t.Fatalf("Console.VisibleWidth() = %d, want %d", got, want)
	}
	if got, want := console.Truncate(styled, 6), Truncate(styled, 6); got != want {
		t.Fatalf("Console.Truncate() = %q, want %q", got, want)
	}
	if got, want := console.TruncateMiddle(styled, 6), TruncateMiddle(styled, 6); got != want {
		t.Fatalf("Console.TruncateMiddle() = %q, want %q", got, want)
	}
	if got, want := console.Wrap(styled, 5), Wrap(styled, 5); got != want {
		t.Fatalf("Console.Wrap() = %q, want %q", got, want)
	}
	if got, want := console.PadRight(styled, 14), PadRight(styled, 14); got != want {
		t.Fatalf("Console.PadRight() = %q, want %q", got, want)
	}
	if got, want := console.PadLeft(styled, 14), PadLeft(styled, 14); got != want {
		t.Fatalf("Console.PadLeft() = %q, want %q", got, want)
	}
	if got, want := console.PadCenter(styled, 14), PadCenter(styled, 14); got != want {
		t.Fatalf("Console.PadCenter() = %q, want %q", got, want)
	}
	if got, want := console.ExpandTabs("a\tb"), ExpandTabs("a\tb"); got != want {
		t.Fatalf("Console.ExpandTabs() = %q, want %q", got, want)
	}
	if got, want := console.Indent("a\nb", "> "), Indent("a\nb", "> "); got != want {
		t.Fatalf("Console.Indent() = %q, want %q", got, want)
	}
}
