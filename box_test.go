package console

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// newBoxTestConsole creates a deterministic console for box rendering tests.
func newBoxTestConsole(width int, unicodeEnabled, colorEnabled bool) (*Console, *bytes.Buffer) {
	output := &bytes.Buffer{}
	console := New(Config{
		Stdout:         output,
		Width:          width,
		UnicodeEnabled: &unicodeEnabled,
		ColorEnabled:   &colorEnabled,
	})
	return console, output
}

// assertBoxVisibleGeometry verifies every physical box line has the requested terminal width.
func assertBoxVisibleGeometry(t *testing.T, rendered string, width int) {
	t.Helper()

	for index, line := range strings.Split(rendered, "\n") {
		if got := VisibleWidth(line); got != width {
			t.Fatalf("box line %d width = %d, want %d: %q", index, got, width, line)
		}
	}
}

// TestRenderBoxRendersUnicodeAndASCII verifies automatic sizing and border fallbacks.
func TestRenderBoxRendersUnicodeAndASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		unicodeEnabled bool
		want           string
	}{
		{
			name:           "Unicode",
			unicodeEnabled: true,
			want: "┌───────┐\n" +
				"│ hello │\n" +
				"└───────┘",
		},
		{
			name:           "ASCII",
			unicodeEnabled: false,
			want: "+-------+\n" +
				"| hello |\n" +
				"+-------+",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			console, _ := newBoxTestConsole(80, test.unicodeEnabled, false)
			got := console.RenderBox("hello", BoxColor(""))
			if got != test.want {
				t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, test.want)
			}
			assertBoxVisibleGeometry(t, got, 9)
		})
	}
}

// TestRenderBoxHonorsWidthPaddingAndTitle verifies fixed dimensions and title placement across wrapped rows.
func TestRenderBoxHonorsWidthPaddingAndTitle(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(80, true, false)
	got := console.RenderBox(
		"abcdefghijk",
		BoxWidth(12),
		BoxPadding(1),
		BoxTitle("Status"),
		BoxColor(""),
	)
	want := "┌─ Status ─┐\n" +
		"│ abcdefgh │\n" +
		"│ ijk      │\n" +
		"└──────────┘"
	if got != want {
		t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, want)
	}
	assertBoxVisibleGeometry(t, got, 12)
}

// TestRenderBoxTruncatesTitleWithoutChangingWidth verifies long titles leave both corners aligned.
func TestRenderBoxTruncatesTitleWithoutChangingWidth(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(80, true, false)
	got := console.RenderBox("x", BoxWidth(8), BoxTitle("abcdef"), BoxColor(""))
	want := "┌─ ab… ┐\n" +
		"│ x    │\n" +
		"└──────┘"
	if got != want {
		t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, want)
	}
	assertBoxVisibleGeometry(t, got, 8)
}

// TestRenderBoxNormalizesAndBalancesTitle verifies title metadata cannot split or restyle the surrounding border.
func TestRenderBoxNormalizesAndBalancesTitle(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\a"
	close := "\x1b]8;;\a"
	console, _ := newBoxTestConsole(80, true, false)
	got := console.RenderBox("x", BoxWidth(20), BoxTitle("alpha\n"+open+"beta"), BoxColor(""))
	if stripped, want := StripANSI(got), "┌─ alpha beta ─────┐\n│ x                │\n└──────────────────┘"; stripped != want {
		t.Fatalf("StripANSI(RenderBox()) =\n%s\nwant:\n%s", stripped, want)
	}
	if !strings.Contains(strings.Split(got, "\n")[0], close+" ") {
		t.Fatalf("RenderBox() did not close its title hyperlink before the border resumed: %q", got)
	}
	assertBoxVisibleGeometry(t, got, 20)
}

// TestRenderBoxNormalizesPadding verifies negative padding becomes zero and oversized padding sets a minimum.
func TestRenderBoxNormalizesPadding(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(80, true, false)
	if got, want := console.RenderBox("ok", BoxPadding(-5), BoxColor("")), "┌──┐\n│ok│\n└──┘"; got != want {
		t.Fatalf("RenderBox(negative padding) = %q, want %q", got, want)
	}

	got := console.RenderBox("a", BoxWidth(2), BoxPadding(3), BoxColor(""))
	want := "┌───────┐\n│   a   │\n└───────┘"
	if got != want {
		t.Fatalf("RenderBox(large padding) = %q, want %q", got, want)
	}
	assertBoxVisibleGeometry(t, got, 9)
}

// TestRenderBoxBoundsAutomaticWidth verifies long content wraps at the console boundary.
func TestRenderBoxBoundsAutomaticWidth(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(10, true, false)
	got := console.RenderBox("abcdefghijklmnop", BoxColor(""))
	want := "┌────────┐\n" +
		"│ abcdef │\n" +
		"│ ghijkl │\n" +
		"│ mnop   │\n" +
		"└────────┘"
	if got != want {
		t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, want)
	}
	assertBoxVisibleGeometry(t, got, 10)
}

// TestRenderBoxBoundsExtremeOptions verifies programmatic dimensions cannot force unsafe allocations.
func TestRenderBoxBoundsExtremeOptions(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(20, true, false)
	got := console.RenderBox("safe", BoxWidth(math.MaxInt), BoxPadding(math.MaxInt), BoxColor(""))
	for index, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width != 20 {
			t.Fatalf("line %d width = %d, want 20: %q", index, width, line)
		}
	}
}

// TestRenderBoxMaintainsNarrowFixedWidth verifies double-width content cannot widen an explicitly sized box.
func TestRenderBoxMaintainsNarrowFixedWidth(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(5, true, false)
	got := console.RenderBox("界", BoxWidth(5), BoxColor(""))
	assertBoxVisibleGeometry(t, got, 5)
}

// TestRenderBoxHandlesWideCombiningAndEmojiText verifies automatic width follows terminal cells rather than bytes.
func TestRenderBoxHandlesWideCombiningAndEmojiText(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(80, true, false)
	got := console.RenderBox("界 e\u0301 👩🏽‍💻", BoxColor(""))
	want := "┌─────────┐\n" +
		"│ 界 e\u0301 👩🏽‍💻 │\n" +
		"└─────────┘"
	if got != want {
		t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, want)
	}
	assertBoxVisibleGeometry(t, got, 11)
}

// TestRenderBoxPreservesMultilineContent verifies CRLF normalization and blank physical rows.
func TestRenderBoxPreservesMultilineContent(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(80, true, false)
	got := console.RenderBox("a\r\n\r\nbb", BoxColor(""))
	want := "┌────┐\n" +
		"│ a  │\n" +
		"│    │\n" +
		"│ bb │\n" +
		"└────┘"
	if got != want {
		t.Fatalf("RenderBox() =\n%s\nwant:\n%s", got, want)
	}
	assertBoxVisibleGeometry(t, got, 6)
}

// TestRenderBoxBalancesStyledContent verifies wrapped ANSI styles cannot color padding or borders.
func TestRenderBoxBalancesStyledContent(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(8, true, true)
	got := console.RenderBox(
		ColorRed+"abcdefgh"+ColorReset,
		BoxWidth(8),
		BoxColor(""),
	)
	if stripped, want := StripANSI(got), "┌──────┐\n│ abcd │\n│ efgh │\n└──────┘"; stripped != want {
		t.Fatalf("StripANSI(RenderBox()) = %q, want %q", stripped, want)
	}
	assertBoxVisibleGeometry(t, got, 8)
	lines := strings.Split(got, "\n")
	for index, line := range lines[1 : len(lines)-1] {
		if !strings.Contains(line, ColorReset+" "+"│") {
			t.Fatalf("content line %d does not reset before right padding and border: %q", index, line)
		}
	}
}

// TestRenderBoxPreservesOSCHyperlinks verifies hyperlink bytes survive wrapping without changing geometry.
func TestRenderBoxPreservesOSCHyperlinks(t *testing.T) {
	t.Parallel()

	open := "\x1b]8;;https://example.test\a"
	close := "\x1b]8;;\a"
	console, _ := newBoxTestConsole(8, true, false)
	got := console.RenderBox(open+"abcdefgh"+close, BoxWidth(8), BoxColor(""))
	if stripped, want := StripANSI(got), "┌──────┐\n│ abcd │\n│ efgh │\n└──────┘"; stripped != want {
		t.Fatalf("StripANSI(RenderBox()) = %q, want %q", stripped, want)
	}
	lines := strings.Split(got, "\n")
	for index, line := range lines[1 : len(lines)-1] {
		if strings.Count(line, open) != 1 || strings.Count(line, close) != 1 {
			t.Fatalf("RenderBox() content line %d does not contain one bounded OSC hyperlink: %q", index, line)
		}
		if !strings.Contains(line, close+" │") {
			t.Fatalf("RenderBox() content line %d does not close its OSC hyperlink before padding: %q", index, line)
		}
	}
	assertBoxVisibleGeometry(t, got, 8)
}

// TestRenderBoxStylesBordersWithoutChangingGeometry verifies optional border color is ANSI-safe.
func TestRenderBoxStylesBordersWithoutChangingGeometry(t *testing.T) {
	t.Parallel()

	console, _ := newBoxTestConsole(5, true, true)
	got := console.RenderBox("x")
	if stripped, want := StripANSI(got), "┌───┐\n│ x │\n└───┘"; stripped != want {
		t.Fatalf("StripANSI(RenderBox()) = %q, want %q", stripped, want)
	}
	if !strings.Contains(got, ColorGray) {
		t.Fatalf("RenderBox() did not style its borders: %q", got)
	}
	assertBoxVisibleGeometry(t, got, 5)
}

// TestBoxWritesExactlyOneTrailingNewline verifies printing adds one delimiter after the rendered box.
func TestBoxWritesExactlyOneTrailingNewline(t *testing.T) {
	t.Parallel()

	console, output := newBoxTestConsole(80, true, false)
	console.Box("ok", nil, BoxColor(""))
	want := "┌────┐\n│ ok │\n└────┘\n"
	if got := output.String(); got != want {
		t.Fatalf("Box() wrote %q, want %q", got, want)
	}
}
