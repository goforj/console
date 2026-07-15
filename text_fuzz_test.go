package console

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzTextLayoutInvariants checks that ANSI removal preserves measured width and bounded helpers stay bounded.
func FuzzTextLayoutInvariants(f *testing.F) {
	f.Add("plain text", 12)
	f.Add(ColorRed+"styled text"+ColorReset, 6)
	f.Add("界e\u0301👩🏽‍💻", 4)
	f.Add("\x1b]8;;https://example.test\ahello\x1b]8;;\a", 3)
	f.Add(string([]byte{'a', 0xff, 'b'}), 2)

	f.Fuzz(func(t *testing.T, value string, requestedWidth int) {
		width := requestedWidth % 80
		if width < 0 {
			width = -width
		}
		width++

		stripped := StripANSI(value)
		if utf8.ValidString(value) && !strings.ContainsRune(stripped, '\x1b') {
			if got, want := VisibleWidth(stripped), VisibleWidth(value); got != want {
				t.Fatalf("VisibleWidth(StripANSI(value)) = %d, want %d for %q", got, want, value)
			}
		}

		truncated := Truncate(value, width)
		if got := VisibleWidth(truncated); got > width {
			t.Fatalf("VisibleWidth(Truncate(value, %d)) = %d for %q", width, got, value)
		}

		for _, line := range strings.Split(Wrap(value, width), "\n") {
			if got := VisibleWidth(line); got > max(width, 2) {
				t.Fatalf("VisibleWidth(Wrap(value, %d) line) = %d for %q", width, got, value)
			}
		}
	})
}
