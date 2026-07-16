package console

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// FuzzTextLayoutInvariants checks that ANSI removal preserves measured width and bounded helpers stay bounded.
func FuzzTextLayoutInvariants(f *testing.F) {
	f.Add("plain text", 12)
	f.Add(ColorRed+"styled text"+ColorReset, 6)
	f.Add("界e\u0301👩🏽‍💻", 4)
	f.Add("A\u200dB각काकी", 1)
	f.Add("🏻A🏻☝🏻", 2)
	f.Add("e"+ColorRed+"\u0301"+ColorReset, 1)
	f.Add("\x1b]8;;https://example.test\ahello\x1b]8;;\a", 3)
	f.Add("\x1bPdevice\x1b[31mdata\x1b\\", 3)
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
		if got := joinDisplayTokens(displayTokens(value)); got != value {
			t.Fatalf("joinDisplayTokens(displayTokens(value)) = %q, want %q", got, value)
		}

		truncated := Truncate(value, width)
		if got := VisibleWidth(truncated); got > width {
			t.Fatalf("VisibleWidth(Truncate(value, %d)) = %d for %q", width, got, value)
		}
		middle := TruncateMiddle(value, width)
		if got := VisibleWidth(middle); got > width {
			t.Fatalf("VisibleWidth(TruncateMiddle(value, %d)) = %d for %q", width, got, value)
		}

		maximum := max(width, maximumDisplayTokenWidth(value))
		for _, line := range strings.Split(Wrap(value, width), "\n") {
			if got := VisibleWidth(line); got > maximum {
				t.Fatalf("VisibleWidth(Wrap(value, %d) line) = %d for %q", width, got, value)
			}
		}
	})
}

// FuzzSanitizeLayoutTextSafety verifies sanitized layout content retains only measured presentation metadata.
func FuzzSanitizeLayoutTextSafety(f *testing.F) {
	f.Add("plain\ntext")
	f.Add(ColorRed + "styled" + ColorReset)
	f.Add("\x1b]8;;https://例.example/文書\x1b\\docs\x1b]8;;\x1b\\")
	f.Add("\x1b]8;;https://example.test/\x1b[31msmuggled\x1b\\")
	f.Add("\x1bPdevice\a\x1b[31mdata\x1b\\")
	f.Add(string([]byte{'a', 0x90, 0xff, 'b'}))

	f.Fuzz(func(t *testing.T, value string) {
		sanitized := sanitizeLayoutText(value, true)
		for index := 0; index < len(sanitized); {
			if sanitized[index] == '\x1b' {
				end, ok := ansiSequenceEnd(sanitized, index)
				if !ok {
					t.Fatalf("sanitizeLayoutText(%q) retained incomplete escape in %q", value, sanitized)
				}
				sequence := sanitized[index:end]
				if !isSGRSequence(sequence) && !safeOSC8Sequence(sequence) {
					t.Fatalf("sanitizeLayoutText(%q) retained unsafe sequence %q", value, sequence)
				}
				index = end
				continue
			}

			runeValue, size := utf8.DecodeRuneInString(sanitized[index:])
			if runeValue == utf8.RuneError && size == 1 {
				if sanitized[index] >= 0x80 && sanitized[index] <= 0x9f {
					t.Fatalf("sanitizeLayoutText(%q) retained C1 byte in %q", value, sanitized)
				}
				index++
				continue
			}
			if unicode.IsControl(runeValue) && runeValue != '\n' && runeValue != '\t' {
				t.Fatalf("sanitizeLayoutText(%q) retained control %U in %q", value, runeValue, sanitized)
			}
			index += size
		}
	})
}

// maximumDisplayTokenWidth returns the largest indivisible width Wrap may place on a line.
func maximumDisplayTokenWidth(value string) int {
	maximum := 0
	for _, token := range displayTokens(value) {
		if token.dynamicTab {
			maximum = max(maximum, tabWidth)
			continue
		}
		maximum = max(maximum, token.width)
	}
	return maximum
}
