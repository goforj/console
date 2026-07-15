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

// TestRenderTableFitsConfiguredWidth verifies oversized columns shrink proportionally and preserve content by wrapping.
func TestRenderTableFitsConfiguredWidth(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(20, true, false)
	got := console.RenderTable([]string{"abcdefghij", "klmnopqrst"}, nil)
	want := "┌────────┬─────────┐\n" +
		"│ abcdef │ klmnopq │\n" +
		"│ ghij   │ rst     │\n" +
		"└────────┴─────────┘"
	if got != want {
		t.Fatalf("RenderTable() =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 20)
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

// TestRenderTablePreservesOSCHyperlinks verifies OSC metadata does not alter measurement or wrapping.
func TestRenderTablePreservesOSCHyperlinks(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\x1b\\"
	close := "\x1b]8;;\x1b\\"
	console, _ := newTableTestConsole(8, true, false)
	got := console.RenderTable([]string{open + "abcdefgh" + close}, nil)
	assertTableVisibleGeometry(t, got, 8)
	if stripped, want := StripANSI(got), "┌──────┐\n│ abcd │\n│ efgh │\n└──────┘"; stripped != want {
		t.Fatalf("StripANSI(RenderTable()) = %q, want %q", stripped, want)
	}
	headerLines := strings.Split(got, "\n")[1:3]
	closeBEL := "\x1b]8;;\a"
	for index, headerLine := range headerLines {
		closeIndex := max(strings.Index(headerLine, closeBEL), strings.Index(headerLine, close))
		paddingIndex := strings.LastIndex(headerLine, " ")
		if closeIndex < 0 || paddingIndex < 0 || closeIndex > paddingIndex {
			t.Fatalf("RenderTable() line %d did not close its OSC hyperlink before padding: %q", index, headerLine)
		}
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

// TestRenderTableCompactUsesOnlyColumnSpacingAndHeaderSeparator verifies the restrained borderless presentation.
func TestRenderTableCompactUsesOnlyColumnSpacingAndHeaderSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		unicodeEnabled bool
		want           string
	}{
		{
			name:           "Unicode",
			unicodeEnabled: true,
			want: "Name   Age\n" +
				"─────  ───\n" +
				"Alice  30\n" +
				"Bob    7",
		},
		{
			name:           "ASCII",
			unicodeEnabled: false,
			want: "Name   Age\n" +
				"-----  ---\n" +
				"Alice  30\n" +
				"Bob    7",
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
				TableCompact(),
			)
			if got != test.want {
				t.Fatalf("RenderTable(TableCompact()) =\n%q\nwant:\n%q", got, test.want)
			}
			if strings.ContainsAny(got, "│|┌┐└┘+") {
				t.Fatalf("RenderTable(TableCompact()) retained frame characters: %q", got)
			}
			for index, line := range strings.Split(got, "\n") {
				if strings.HasSuffix(line, " ") {
					t.Fatalf("compact table line %d has trailing padding: %q", index, line)
				}
			}
		})
	}
}

// TestRenderTableCompactSupportsRowsWithoutHeaders verifies the separator belongs only to a real header.
func TestRenderTableCompactSupportsRowsWithoutHeaders(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(nil, [][]string{{"a", "bb"}, {"long", "c"}}, TableCompact())
	want := "a     bb\n" +
		"long  c"
	if got != want {
		t.Fatalf("RenderTable(TableCompact()) =\n%q\nwant:\n%q", got, want)
	}
}

// TestRenderTableWidthsWrapsCells verifies explicit widths control geometry without discarding content.
func TestRenderTableWidthsWrapsCells(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(
		[]string{"Name", "Note"},
		[][]string{{"界", "abcdefgh"}},
		TableWidths(6, 4),
	)
	want := "┌────────┬──────┐\n" +
		"│ Name   │ Note │\n" +
		"├────────┼──────┤\n" +
		"│ 界     │ abcd │\n" +
		"│        │ efgh │\n" +
		"└────────┴──────┘"
	if got != want {
		t.Fatalf("RenderTable(TableWidths()) =\n%s\nwant:\n%s", got, want)
	}
	assertTableVisibleGeometry(t, got, 17)
}

// TestRenderTableWidthsLeavesNonpositiveColumnsAutomatic verifies callers can configure only selected columns.
func TestRenderTableWidthsLeavesNonpositiveColumnsAutomatic(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	headers := []string{"Name", "State"}
	rows := [][]string{{"api", "ready"}}
	want := console.RenderTable(headers, rows)
	for _, widths := range [][]int{{0}, {-1, 0}, {0, -10, 100}} {
		if got := console.RenderTable(headers, rows, TableWidths(widths...)); got != want {
			t.Fatalf("RenderTable(TableWidths(%v)) = %q, want automatic %q", widths, got, want)
		}
	}
}

// TestRenderTableConfiguredWidthsStillFitConsole verifies fixed requests remain subject to terminal geometry.
func TestRenderTableConfiguredWidthsStillFitConsole(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(10, true, false)
	got := console.RenderTable(
		[]string{"First", "Second"},
		[][]string{{"abcdefghij", "klmnopqrst"}},
		TableWidths(20, 20),
	)
	assertTableVisibleGeometry(t, got, 10)
	for _, value := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "kl", "mn", "op", "qr", "st"} {
		if !strings.Contains(got, value) {
			t.Fatalf("RenderTable(TableWidths()) lost wrapped content %q: %q", value, got)
		}
	}
}

// TestRenderCompactTableOptionsStayWithinConsoleWidth verifies borderless tables retain their width contract after composition.
func TestRenderCompactTableOptionsStayWithinConsoleWidth(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(8, true, false)
	got := console.RenderTable(
		[]string{"First", "Second"},
		[][]string{{"abcdefghij", "klmnopqrst"}},
		TableCompact(),
		TableWidths(20, 20),
		TableCenterAlign(0),
		TableRightAlign(1),
	)
	for index, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width > console.Width() {
			t.Fatalf("compact line %d width = %d, maximum %d: %q", index, width, console.Width(), line)
		}
		if strings.HasSuffix(line, " ") {
			t.Fatalf("compact line %d retained trailing padding: %q", index, line)
		}
	}
	for _, fragment := range []string{"Fir", "st", "abc", "ghi", "Sec", "ond", "klm", "qrs"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("compact table lost wrapped content %q: %q", fragment, got)
		}
	}
}

// TestRenderTableAlignsANSIAndWideCells verifies alignment is based on terminal cells rather than bytes.
func TestRenderTableAlignsANSIAndWideCells(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	styledWide := ColorRed + "界" + ColorReset
	got := console.RenderTable(
		[]string{"Name", "Count"},
		[][]string{{styledWide, "7"}},
		TableWidths(6, 5),
		TableCenterAlign(0),
		TableRightAlign(1),
	)
	want := "┌────────┬───────┐\n" +
		"│  Name  │ Count │\n" +
		"├────────┼───────┤\n" +
		"│   界   │     7 │\n" +
		"└────────┴───────┘"
	if stripped := StripANSI(got); stripped != want {
		t.Fatalf("StripANSI(RenderTable(aligned)) =\n%s\nwant:\n%s", stripped, want)
	}
	if !strings.Contains(got, styledWide) {
		t.Fatalf("RenderTable(aligned) discarded caller styling: %q", got)
	}
	assertTableVisibleGeometry(t, got, 18)
}

// TestRenderTableAlignmentOptionsIgnoreInvalidColumnsAndUseLastOption verifies predictable option composition.
func TestRenderTableAlignmentOptionsIgnoreInvalidColumnsAndUseLastOption(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(
		[]string{"A", "B"},
		[][]string{{"x", "y"}},
		TableWidths(3, 3),
		TableRightAlign(-1, 0, 20),
		TableCenterAlign(0),
	)
	want := "┌─────┬─────┐\n" +
		"│  A  │ B   │\n" +
		"├─────┼─────┤\n" +
		"│  x  │ y   │\n" +
		"└─────┴─────┘"
	if got != want {
		t.Fatalf("RenderTable(composed alignment) =\n%s\nwant:\n%s", got, want)
	}
}

// TestTableOptionsCloneCallerSlices verifies later caller mutation cannot alter reusable options.
func TestTableOptionsCloneCallerSlices(t *testing.T) {
	t.Parallel()

	widths := []int{3, 3}
	rightColumns := []int{1}
	centerColumns := []int{0}
	widthOption := TableWidths(widths...)
	rightOption := TableRightAlign(rightColumns...)
	centerOption := TableCenterAlign(centerColumns...)
	widths[0] = 30
	rightColumns[0] = 0
	centerColumns[0] = 1

	console, _ := newTableTestConsole(80, true, false)
	got := console.RenderTable(
		[]string{"A", "B"},
		[][]string{{"x", "y"}},
		widthOption,
		rightOption,
		centerOption,
	)
	want := "┌─────┬─────┐\n" +
		"│  A  │   B │\n" +
		"├─────┼─────┤\n" +
		"│  x  │   y │\n" +
		"└─────┴─────┘"
	if got != want {
		t.Fatalf("RenderTable(reused options) =\n%s\nwant:\n%s", got, want)
	}
}

// TestRenderTableIgnoresNilOptions verifies optional composition remains nil-safe.
func TestRenderTableIgnoresNilOptions(t *testing.T) {
	t.Parallel()

	console, _ := newTableTestConsole(80, true, false)
	headers := []string{"A"}
	rows := [][]string{{"x"}}
	if got, want := console.RenderTable(headers, rows, nil), console.RenderTable(headers, rows); got != want {
		t.Fatalf("RenderTable(nil option) = %q, want %q", got, want)
	}
}

// TestTableWritesConfiguredPresentation verifies printing forwards options and adds one newline.
func TestTableWritesConfiguredPresentation(t *testing.T) {
	t.Parallel()

	console, output := newTableTestConsole(80, true, false)
	console.Table(
		[]string{"Name", "Count"},
		[][]string{{"api", "2"}},
		TableCompact(),
		TableRightAlign(1),
	)
	want := "Name  Count\n" +
		"────  ─────\n" +
		"api       2\n"
	if got := output.String(); got != want {
		t.Fatalf("Table(options) wrote %q, want %q", got, want)
	}
}
