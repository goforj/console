package console

import (
	"bytes"
	"strings"
	"testing"
)

// newTableTestConsole creates a deterministic console for table rendering tests.
func newTableTestConsole(width int, unicodeEnabled, colorEnabled bool) (*Console, *bytes.Buffer) {
	output := &bytes.Buffer{}
	console := New(Config{
		Stdout:         output,
		Width:          width,
		UnicodeEnabled: &unicodeEnabled,
		ColorEnabled:   &colorEnabled,
	})
	return console, output
}

// assertTableVisibleGeometry verifies every physical row and border has one terminal width.
func assertTableVisibleGeometry(t *testing.T, rendered string, width int) {
	t.Helper()

	for index, line := range strings.Split(rendered, "\n") {
		if got := VisibleWidth(line); got != width {
			t.Fatalf("table line %d width = %d, want %d: %q", index, got, width, line)
		}
	}
}

// TestRenderTableReturnsEmptyWithoutColumns verifies empty and zero-cell inputs have no frame.
func TestRenderTableReturnsEmptyWithoutColumns(t *testing.T) {
	t.Parallel()

	console, output := newTableTestConsole(80, true, false)
	tests := []struct {
		name    string
		headers []string
		rows    [][]string
	}{
		{name: "nil"},
		{name: "empty slices", headers: []string{}, rows: [][]string{}},
		{name: "empty row", rows: [][]string{{}}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := console.RenderTable(test.headers, test.rows); got != "" {
				t.Fatalf("RenderTable() = %q, want empty output", got)
			}
		})
	}
	console.Table(nil, nil)
	if got := output.String(); got != "" {
		t.Fatalf("Table(nil, nil) wrote %q, want no output", got)
	}
}

// TestRenderTableRendersUnicodeAndASCII verifies headers, data, padding, and border fallbacks.
func TestRenderTableRendersUnicodeAndASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		unicodeEnabled bool
		want           string
	}{
		{
			name:           "Unicode",
			unicodeEnabled: true,
			want: "┌───────┬─────┐\n" +
				"│ Name  │ Age │\n" +
				"├───────┼─────┤\n" +
				"│ Alice │ 30  │\n" +
				"│ Bob   │ 7   │\n" +
				"└───────┴─────┘",
		},
		{
			name:           "ASCII",
			unicodeEnabled: false,
			want: "+-------+-----+\n" +
				"| Name  | Age |\n" +
				"+-------+-----+\n" +
				"| Alice | 30  |\n" +
				"| Bob   | 7   |\n" +
				"+-------+-----+",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			console, _ := newTableTestConsole(80, test.unicodeEnabled, false)
			got := console.RenderTable(
				[]string{"Name", "Age"},
				[][]string{{"Alice", "30"}, {"Bob", "7"}},
			)
			if got != test.want {
				t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, test.want)
			}
			assertTableVisibleGeometry(t, got, 15)
		})
	}
}

// TestRenderTableNormalizesRaggedMultilineRows verifies missing cells and physical row heights align.
func TestRenderTableNormalizesRaggedMultilineRows(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(
		[]string{"A", "B", "C"},
		[][]string{
			{"one\n二", "x"},
			{"z", "p\nq\nr", "tail"},
		},
	)
	want := "┌─────┬───┬──────┐\n" +
		"│ A   │ B │ C    │\n" +
		"├─────┼───┼──────┤\n" +
		"│ one │ x │      │\n" +
		"│ 二  │   │      │\n" +
		"│ z   │ p │ tail │\n" +
		"│     │ q │      │\n" +
		"│     │ r │      │\n" +
		"└─────┴───┴──────┘"
	if got != want {
		t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 18)
}

// TestRenderTableSupportsRowsWithoutHeaders verifies ragged data can establish every column.
func TestRenderTableSupportsRowsWithoutHeaders(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(nil, [][]string{{"a"}, {"bb", "cc", "d"}})
	want := "┌────┬────┬───┐\n" +
		"│ a  │    │   │\n" +
		"│ bb │ cc │ d │\n" +
		"└────┴────┴───┘"
	if got != want {
		t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 15)
}

// TestRenderTableFitsConfiguredWidth verifies oversized columns shrink proportionally and truncate visibly.
func TestRenderTableFitsConfiguredWidth(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(20, true, false)
	got := console.RenderTable([]string{"abcdefghij", "klmnopqrst"}, nil)
	assertTableVisibleGeometry(t, got, 20)
	if !strings.Contains(got, "abcde…") {
		t.Fatalf("RenderTable() did not truncate the first column: %q", got)
	}
	if !strings.Contains(got, "klmnop…") {
		t.Fatalf("RenderTable() did not truncate the second column: %q", got)
	}
}

// TestRenderTableUsesStructuralMinimum verifies very narrow terminals retain a valid frame.
func TestRenderTableUsesStructuralMinimum(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(1, true, false)
	got := console.RenderTable([]string{"a", "b"}, [][]string{{"c", "d"}})
	assertTableVisibleGeometry(t, got, 9)
	if stripped, want := StripANSI(got), "┌───┬───┐\n│ a │ b │\n├───┼───┤\n│ c │ d │\n└───┴───┘"; stripped != want {
		t.Fatalf("RenderTable() = %q, want %q", stripped, want)
	}
}

// TestRenderTableTruncatesWideGlyphsAtNarrowWidths verifies two-cell glyphs cannot deform one-cell columns.
func TestRenderTableTruncatesWideGlyphsAtNarrowWidths(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(5, true, false)
	got := console.RenderTable([]string{"界"}, [][]string{{"👩🏽‍💻"}})
	want := "┌───┐\n" +
		"│ … │\n" +
		"├───┤\n" +
		"│ … │\n" +
		"└───┘"
	if got != want {
		t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 5)
}

// TestRenderTableUsesASCIIFallbackForTruncation verifies ASCII mode does not emit a Unicode ellipsis.
func TestRenderTableUsesASCIIFallbackForTruncation(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(5, false, false)
	got := console.RenderTable([]string{"界"}, [][]string{{"👩🏽‍💻"}})
	want := "+---+\n" +
		"| . |\n" +
		"+---+\n" +
		"| . |\n" +
		"+---+"
	if got != want {
		t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 5)
}

// TestRenderTableMeasuresANSIWideCombiningAndEmojiCells verifies rich text shares equal visible geometry.
func TestRenderTableMeasuresANSIWideCombiningAndEmojiCells(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(
		[]string{ColorRed + "Key" + ColorReset, "値"},
		[][]string{{"e\u0301", "👩🏽‍💻"}, {"界", "ok"}},
	)
	want := "┌─────┬────┐\n" +
		"│ Key │ 値 │\n" +
		"├─────┼────┤\n" +
		"│ e\u0301   │ 👩🏽‍💻 │\n" +
		"│ 界  │ ok │\n" +
		"└─────┴────┘"
	if stripped := StripANSI(got); stripped != want {
		t.Fatalf("StripANSI(RenderTable()) =\n%s\nwant:\n%s", stripped, want)
	}
	if !strings.Contains(got, ColorRed+"Key"+ColorReset) {
		t.Fatalf("RenderTable() discarded caller styling: %q", got)
	}
	assertTableVisibleGeometry(t, got, 12)
}

// TestRenderTableExpandsTabsBeforeEmbedding verifies absolute cursor columns cannot deform borders.
func TestRenderTableExpandsTabsBeforeEmbedding(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(nil, [][]string{{"\t"}})
	want := "┌──────────┐\n" +
		"│          │\n" +
		"└──────────┘"
	if got != want {
		t.Fatalf("RenderTable(tab) =\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "\t") {
		t.Fatalf("RenderTable(tab) retained a cursor-relative tab: %q", got)
	}
	assertTableVisibleGeometry(t, got, 12)
}

// TestRenderTablePreservesOSCHyperlinks verifies OSC metadata does not alter measurement or truncation.
func TestRenderTablePreservesOSCHyperlinks(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\x1b\\"
	close := "\x1b]8;;\x1b\\"
	console, _ := newTableTestConsole(8, true, false)
	got := console.RenderTable([]string{open + "abcdefgh" + close}, nil)
	assertTableVisibleGeometry(t, got, 8)
	if stripped, want := StripANSI(got), "┌──────┐\n│ abc… │\n└──────┘"; stripped != want {
		t.Fatalf("StripANSI(RenderTable()) = %q, want %q", stripped, want)
	}
	headerLine := strings.Split(got, "\n")[1]
	closeBEL := "\x1b]8;;\a"
	closeIndex := strings.Index(headerLine, closeBEL)
	ellipsisIndex := strings.Index(headerLine, "…")
	if closeIndex < 0 || ellipsisIndex < 0 || closeIndex > ellipsisIndex {
		t.Fatalf("RenderTable() did not close its OSC hyperlink before the ellipsis and padding: %q", headerLine)
	}
}

// TestRenderTableStylesHeadersAndBordersWithoutChangingGeometry verifies ANSI policy is layout-neutral.
func TestRenderTableStylesHeadersAndBordersWithoutChangingGeometry(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, true)
	got := console.RenderTable([]string{"A"}, [][]string{{"x"}})
	if stripped, want := StripANSI(got), "┌───┐\n│ A │\n├───┤\n│ x │\n└───┘"; stripped != want {
		t.Fatalf("StripANSI(RenderTable()) = %q, want %q", stripped, want)
	}
	if !strings.Contains(got, StyleBold+"A"+ColorReset) {
		t.Fatalf("RenderTable() did not style its header: %q", got)
	}
	if !strings.Contains(got, ColorGray) {
		t.Fatalf("RenderTable() did not style its borders: %q", got)
	}
	assertTableVisibleGeometry(t, got, 5)
}

// TestTableWritesExactlyOneTrailingNewline verifies printing adds one delimiter after a nonempty table.
func TestTableWritesExactlyOneTrailingNewline(t *testing.T) {
	t.Parallel()

	console, output := newTableTestConsole(80, true, false)
	console.Table([]string{"A"}, [][]string{{"x"}})
	want := "┌───┐\n│ A │\n├───┤\n│ x │\n└───┘\n"
	if got := output.String(); got != want {
		t.Fatalf("Table() wrote %q, want %q", got, want)
	}
}
