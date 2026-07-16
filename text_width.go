package console

import "unicode/utf8"

// codePointRange describes one inclusive Unicode property interval.
type codePointRange struct {
	first rune
	last  rune
}

// emojiModifierBaseRanges mirrors Unicode 15 Emoji_Modifier_Base, matching uniseg's data version.
var emojiModifierBaseRanges = [...]codePointRange{
	{0x261d, 0x261d},
	{0x26f9, 0x26f9},
	{0x270a, 0x270d},
	{0x1f385, 0x1f385},
	{0x1f3c2, 0x1f3c4},
	{0x1f3c7, 0x1f3c7},
	{0x1f3ca, 0x1f3cc},
	{0x1f442, 0x1f443},
	{0x1f446, 0x1f450},
	{0x1f466, 0x1f478},
	{0x1f47c, 0x1f47c},
	{0x1f481, 0x1f483},
	{0x1f485, 0x1f487},
	{0x1f48f, 0x1f48f},
	{0x1f491, 0x1f491},
	{0x1f4aa, 0x1f4aa},
	{0x1f574, 0x1f575},
	{0x1f57a, 0x1f57a},
	{0x1f590, 0x1f590},
	{0x1f595, 0x1f596},
	{0x1f645, 0x1f647},
	{0x1f64b, 0x1f64f},
	{0x1f6a3, 0x1f6a3},
	{0x1f6b4, 0x1f6b6},
	{0x1f6c0, 0x1f6c0},
	{0x1f6cc, 0x1f6cc},
	{0x1f90c, 0x1f90c},
	{0x1f90f, 0x1f90f},
	{0x1f918, 0x1f91f},
	{0x1f926, 0x1f926},
	{0x1f930, 0x1f939},
	{0x1f93c, 0x1f93e},
	{0x1f977, 0x1f977},
	{0x1f9b5, 0x1f9b6},
	{0x1f9b8, 0x1f9b9},
	{0x1f9bb, 0x1f9bb},
	{0x1f9cd, 0x1f9cf},
	{0x1f9d1, 0x1f9dd},
	{0x1fac3, 0x1fac5},
	{0x1faf0, 0x1faf8},
}

// normalizedClusterWidth corrects context-sensitive emoji modifiers that UAX #29 groups as Extend.
func normalizedClusterWidth(cluster string, width int) int {
	firstRune, _ := utf8.DecodeRuneInString(cluster)
	modifierCount := 0
	for _, runeValue := range cluster {
		if isEmojiModifier(runeValue) {
			modifierCount++
		}
	}
	if modifierCount == 0 {
		return width
	}
	if isEmojiModifierBase(firstRune) {
		return max(width, 2)
	}
	return width + modifierCount*2
}

// isEmojiModifier reports whether runeValue is one of the five Fitzpatrick modifiers.
func isEmojiModifier(runeValue rune) bool {
	return runeValue >= 0x1f3fb && runeValue <= 0x1f3ff
}

// isEmojiModifierBase reports whether runeValue accepts an immediately following emoji modifier.
func isEmojiModifierBase(runeValue rune) bool {
	low := 0
	high := len(emojiModifierBaseRanges)
	for low < high {
		middle := low + (high-low)/2
		candidate := emojiModifierBaseRanges[middle]
		if runeValue < candidate.first {
			high = middle
			continue
		}
		if runeValue > candidate.last {
			low = middle + 1
			continue
		}
		return true
	}
	return false
}
