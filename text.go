package console

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const tabWidth = 8

// StripANSI removes complete ANSI CSI, OSC, and ESC sequences from value.
// Incomplete escape sequences are retained so malformed input is not silently discarded.
// @group Text
func StripANSI(value string) string {
	var output strings.Builder
	for index := 0; index < len(value); {
		if end, ok := ansiSequenceEnd(value, index); ok {
			index = end
			continue
		}
		_, size := utf8.DecodeRuneInString(value[index:])
		output.WriteString(value[index : index+size])
		index += size
	}
	return output.String()
}

// StripANSI removes complete ANSI sequences from value through a Console instance.
// Its behavior matches the package-level StripANSI helper.
// @group Text
func (c *Console) StripANSI(value string) string {
	return StripANSI(value)
}

// VisibleWidth returns the largest terminal-cell width among value's lines.
// ANSI escapes and combining characters occupy no cells, tabs advance to an eight-cell stop,
// and common East Asian and emoji runes occupy two cells.
// @group Text
func VisibleWidth(value string) int {
	maximum := 0
	current := 0
	for _, token := range displayTokens(value) {
		if token.newline {
			maximum = max(maximum, current)
			current = 0
			continue
		}
		current += tokenWidthAt(token, current)
	}
	return max(maximum, current)
}

// VisibleWidth returns value's terminal-cell width through a Console instance.
// Its behavior matches the package-level VisibleWidth helper.
// @group Text
func (c *Console) VisibleWidth(value string) int {
	return VisibleWidth(value)
}

// Truncate shortens each line of value to width terminal cells and uses an ellipsis when content is removed.
// Active SGR styles and OSC 8 hyperlinks are closed before the ellipsis.
// Values less than one produce an empty string.
// @group Text
func Truncate(value string, width int) string {
	return truncateWithTail(value, width, "…")
}

// Truncate shortens value through a Console instance using the public Unicode ellipsis contract.
// Its behavior matches the package-level Truncate helper regardless of the console's layout policy.
// @group Text
func (c *Console) Truncate(value string, width int) string {
	return Truncate(value, width)
}

// TruncateMiddle shortens each line of value to width terminal cells by replacing its center with an ellipsis.
// Active SGR styles and OSC 8 hyperlinks are kept with the visible text on either side of the ellipsis.
// Values less than one produce an empty string.
// @group Text
func TruncateMiddle(value string, width int) string {
	if width < 1 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for index, line := range lines {
		lines[index] = truncateMiddleLine(line, width)
	}
	return strings.Join(lines, "\n")
}

// TruncateMiddle shortens value through a Console instance using the public Unicode ellipsis contract.
// Its behavior matches the package-level TruncateMiddle helper regardless of the console's layout policy.
// @group Text
func (c *Console) TruncateMiddle(value string, width int) string {
	return TruncateMiddle(value, width)
}

// truncateWithTail shortens each line with a caller-selected one-cell truncation marker.
func truncateWithTail(value string, width int, tail string) string {
	if width < 1 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for index, line := range lines {
		lines[index] = truncateLine(line, width, tail)
	}
	return strings.Join(lines, "\n")
}

// truncate selects the presentation marker that matches the console's Unicode policy.
func (c *Console) truncate(value string, width int) string {
	if c.unicodeEnabled {
		return Truncate(value, width)
	}
	return truncateWithTail(value, width, ".")
}

// Wrap inserts newlines so each resulting line fits within width terminal cells where possible.
// Existing line breaks and ANSI styling are preserved; active SGR styles and OSC 8 hyperlinks are balanced
// at each line boundary so they cannot bleed into surrounding layout. Long unbroken words wrap at cell boundaries.
// Breakable whitespace at the beginning or end of a resulting line is removed.
// Values less than one are returned unchanged.
// @group Text
func Wrap(value string, width int) string {
	if width < 1 || value == "" {
		return value
	}
	lines := wrapLines(strings.ReplaceAll(value, "\r\n", "\n"), width)
	return strings.Join(balanceANSILines(lines), "\n")
}

// Wrap inserts terminal-cell-aware line breaks through a Console instance.
// Its behavior matches the package-level Wrap helper.
// @group Text
func (c *Console) Wrap(value string, width int) string {
	return Wrap(value, width)
}

// PadRight appends spaces until every line reaches width terminal cells.
// Lines already at or beyond width are unchanged.
// @group Text
func PadRight(value string, width int) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for index, line := range lines {
		padding := width - VisibleWidth(line)
		if padding > 0 {
			lines[index] = line + strings.Repeat(" ", padding)
		}
	}
	return strings.Join(lines, "\n")
}

// PadRight pads value to a terminal-cell width through a Console instance.
// Its behavior matches the package-level PadRight helper.
// @group Text
func (c *Console) PadRight(value string, width int) string {
	return PadRight(value, width)
}

// PadLeft prepends spaces until every line reaches width terminal cells.
// Lines already at or beyond width are unchanged. Tabs are expanded only on lines that need padding
// because leading spaces otherwise change their terminal tab stops.
// @group Text
func PadLeft(value string, width int) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for index, line := range lines {
		if VisibleWidth(line) >= width {
			continue
		}
		line = ExpandTabs(line)
		lines[index] = strings.Repeat(" ", width-VisibleWidth(line)) + line
	}
	return strings.Join(lines, "\n")
}

// PadLeft pads value on the left through a Console instance.
// Its behavior matches the package-level PadLeft helper.
// @group Text
func (c *Console) PadLeft(value string, width int) string {
	return PadLeft(value, width)
}

// PadCenter adds spaces around every line until it reaches width terminal cells.
// Odd padding places the extra space on the right. Lines already at or beyond width are unchanged.
// Tabs are expanded only on lines that need padding so their alignment remains stable.
// @group Text
func PadCenter(value string, width int) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for index, line := range lines {
		if VisibleWidth(line) >= width {
			continue
		}
		line = ExpandTabs(line)
		padding := width - VisibleWidth(line)
		left := padding / 2
		lines[index] = strings.Repeat(" ", left) + line + strings.Repeat(" ", padding-left)
	}
	return strings.Join(lines, "\n")
}

// PadCenter pads value on both sides through a Console instance.
// Its behavior matches the package-level PadCenter helper.
// @group Text
func (c *Console) PadCenter(value string, width int) string {
	return PadCenter(value, width)
}

// ExpandTabs replaces tabs with spaces at eight-cell stops on each line.
// ANSI escape sequences do not affect tab positions.
// @group Text
func ExpandTabs(value string) string {
	if !strings.Contains(value, "\t") {
		return value
	}

	var output strings.Builder
	current := 0
	for _, token := range displayTokens(value) {
		if token.newline {
			output.WriteString(token.raw)
			current = 0
			continue
		}
		width := tokenWidthAt(token, current)
		if token.dynamicTab {
			output.WriteString(strings.Repeat(" ", width))
		} else {
			output.WriteString(token.raw)
		}
		current += width
	}
	return output.String()
}

// ExpandTabs replaces tabs through a Console instance.
// Its behavior matches the package-level ExpandTabs helper.
// @group Text
func (c *Console) ExpandTabs(value string) string {
	return ExpandTabs(value)
}

// Indent prefixes every line in value with prefix.
// Empty input remains empty.
// @group Text
func Indent(value, prefix string) string {
	if value == "" {
		return ""
	}
	return prefix + strings.ReplaceAll(value, "\n", "\n"+prefix)
}

// Indent prefixes each line through a Console instance.
// Its behavior matches the package-level Indent helper.
// @group Text
func (c *Console) Indent(value, prefix string) string {
	return Indent(value, prefix)
}

// sanitizeLayoutText preserves printable content, SGR styling, and OSC 8 links while removing terminal controls
// that can move the cursor, mutate terminal state, or invalidate measured geometry.
func sanitizeLayoutText(value string, multiline bool) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	var output strings.Builder
	for index := 0; index < len(value); {
		if end, ok := ansiSequenceEnd(value, index); ok {
			sequence := value[index:end]
			if isSGRSequence(sequence) {
				output.WriteString(sequence)
			} else if _, hyperlink := osc8Target(sequence); hyperlink {
				output.WriteString(sequence)
			}
			index = end
			continue
		}

		runeValue, size := utf8.DecodeRuneInString(value[index:])
		raw := value[index : index+size]
		index += size
		switch runeValue {
		case '\n', '\r':
			if multiline {
				output.WriteByte('\n')
			} else {
				output.WriteByte(' ')
			}
		case '\t':
			output.WriteByte('\t')
		default:
			if !unicode.IsControl(runeValue) {
				output.WriteString(raw)
			}
		}
	}
	return output.String()
}

// singleLineLayoutText normalizes metadata fields and closes caller styling before generated layout resumes.
func singleLineLayoutText(value string) string {
	value = ExpandTabs(sanitizeLayoutText(value, false))
	return balanceANSILines([]string{value})[0]
}

// displayToken contains one escape, rune, or logical newline and its layout properties.
type displayToken struct {
	raw        string
	value      rune
	width      int
	space      bool
	newline    bool
	escape     bool
	dynamicTab bool
}

// displayTokens tokenizes styled text while retaining the exact bytes needed to rebuild it.
func displayTokens(value string) []displayToken {
	tokens := make([]displayToken, 0, len(value))
	joinNext := false
	lastBaseToken := -1
	regionalPending := false

	for index := 0; index < len(value); {
		if end, ok := ansiSequenceEnd(value, index); ok {
			tokens = append(tokens, displayToken{raw: value[index:end], escape: true})
			index = end
			continue
		}

		runeValue, size := utf8.DecodeRuneInString(value[index:])
		raw := value[index : index+size]
		index += size

		if runeValue == '\n' {
			tokens = append(tokens, displayToken{raw: raw, newline: true})
			joinNext = false
			lastBaseToken = -1
			regionalPending = false
			continue
		}
		if runeValue == '\r' {
			tokens = append(tokens, displayToken{raw: raw})
			lastBaseToken = -1
			continue
		}
		if runeValue == '\t' {
			tokens = append(tokens, displayToken{raw: raw, space: true, dynamicTab: true})
			lastBaseToken = -1
			regionalPending = false
			continue
		}
		if runeValue == '\ufe0f' {
			if lastBaseToken >= 0 && tokens[lastBaseToken].width == 1 && !tokens[lastBaseToken].space {
				tokens[lastBaseToken].width = 2
			}
			tokens = append(tokens, displayToken{raw: raw})
			continue
		}
		if runeValue == '\ufe0e' {
			if lastBaseToken >= 0 && tokens[lastBaseToken].width == 2 &&
				isEmojiPresentationRune(tokens[lastBaseToken].value) {
				tokens[lastBaseToken].width = 1
			}
			tokens = append(tokens, displayToken{raw: raw})
			continue
		}

		baseWidth := runeCellWidth(runeValue)
		width := baseWidth
		if joinNext && width > 0 {
			width = 0
			joinNext = false
		}
		if runeValue == '\u200d' {
			joinNext = true
		}

		if isRegionalIndicator(runeValue) {
			if regionalPending {
				width = 0
				regionalPending = false
			} else {
				width = 2
				regionalPending = true
			}
		} else if width > 0 {
			regionalPending = false
		}

		tokens = append(tokens, displayToken{
			raw:   raw,
			value: runeValue,
			width: width,
			space: isBreakableSpace(runeValue),
		})
		if baseWidth > 0 {
			lastBaseToken = len(tokens) - 1
		}
	}
	return tokens
}

// ansiSequenceEnd returns the exclusive end of one complete escape sequence at start.
func ansiSequenceEnd(value string, start int) (int, bool) {
	if start >= len(value) || value[start] != '\x1b' || start+1 >= len(value) {
		return start, false
	}

	switch value[start+1] {
	case '[':
		for index := start + 2; index < len(value); index++ {
			if value[index] >= 0x40 && value[index] <= 0x7e {
				return index + 1, true
			}
			if value[index] < 0x20 || value[index] > 0x3f {
				return start, false
			}
		}
		return start, false
	case ']':
		for index := start + 2; index < len(value); index++ {
			if value[index] == '\a' {
				return index + 1, true
			}
			if value[index] == '\r' || value[index] == '\n' {
				return start, false
			}
			if value[index] == '\x1b' && index+1 < len(value) && value[index+1] == '\\' {
				return index + 2, true
			}
		}
		return start, false
	default:
		index := start + 1
		for index < len(value) && value[index] >= 0x20 && value[index] <= 0x2f {
			index++
		}
		if index < len(value) && value[index] >= 0x30 && value[index] <= 0x7e {
			return index + 1, true
		}
		return start, false
	}
}

// runeCellWidth approximates wcwidth with standard-library Unicode tables and stable wide ranges.
func runeCellWidth(runeValue rune) int {
	if runeValue == 0 || runeValue < 0x20 || runeValue >= 0x7f && runeValue < 0xa0 {
		return 0
	}
	if runeValue == '\u200d' || runeValue >= 0xfe00 && runeValue <= 0xfe0f ||
		runeValue >= 0xe0100 && runeValue <= 0xe01ef ||
		runeValue >= 0x1f3fb && runeValue <= 0x1f3ff ||
		unicode.Is(unicode.Mn, runeValue) || unicode.Is(unicode.Me, runeValue) || unicode.Is(unicode.Cf, runeValue) {
		return 0
	}
	if isWideRune(runeValue) {
		return 2
	}
	return 1
}

// isBreakableSpace keeps non-breaking Unicode separators attached to their surrounding text.
func isBreakableSpace(runeValue rune) bool {
	switch runeValue {
	case '\u00a0', '\u2007', '\u202f':
		return false
	default:
		return unicode.IsSpace(runeValue)
	}
}

// isRegionalIndicator reports whether runeValue participates in a two-rune flag glyph.
func isRegionalIndicator(runeValue rune) bool {
	return runeValue >= 0x1f1e6 && runeValue <= 0x1f1ff
}

// isWideRune covers the stable East Asian and emoji ranges commonly rendered as two cells.
func isWideRune(runeValue rune) bool {
	return runeValue >= 0x1100 && runeValue <= 0x115f ||
		runeValue == 0x2329 || runeValue == 0x232a ||
		isEmojiPresentationRune(runeValue) ||
		runeValue >= 0x2e80 && runeValue <= 0x303e ||
		runeValue >= 0x3040 && runeValue <= 0xa4cf ||
		runeValue >= 0xac00 && runeValue <= 0xd7a3 ||
		runeValue >= 0xf900 && runeValue <= 0xfaff ||
		runeValue >= 0xfe10 && runeValue <= 0xfe19 ||
		runeValue >= 0xfe30 && runeValue <= 0xfe6f ||
		runeValue >= 0xff00 && runeValue <= 0xff60 ||
		runeValue >= 0xffe0 && runeValue <= 0xffe6 ||
		runeValue >= 0x1f000 && runeValue <= 0x1faff ||
		runeValue >= 0x20000 && runeValue <= 0x3fffd
}

// isEmojiPresentationRune covers symbols whose default Unicode presentation commonly occupies two terminal cells.
func isEmojiPresentationRune(runeValue rune) bool {
	return runeValue >= 0x231a && runeValue <= 0x231b ||
		runeValue >= 0x23e9 && runeValue <= 0x23ec ||
		runeValue == 0x23f0 || runeValue == 0x23f3 ||
		runeValue >= 0x25fd && runeValue <= 0x25fe ||
		runeValue >= 0x2614 && runeValue <= 0x2615 ||
		runeValue >= 0x2648 && runeValue <= 0x2653 ||
		runeValue == 0x267f || runeValue == 0x2693 || runeValue == 0x26a1 ||
		runeValue >= 0x26aa && runeValue <= 0x26ab ||
		runeValue >= 0x26bd && runeValue <= 0x26be ||
		runeValue >= 0x26c4 && runeValue <= 0x26c5 ||
		runeValue == 0x26ce || runeValue == 0x26d4 || runeValue == 0x26ea ||
		runeValue >= 0x26f2 && runeValue <= 0x26f3 ||
		runeValue == 0x26f5 || runeValue == 0x26fa || runeValue == 0x26fd ||
		runeValue == 0x2705 || runeValue >= 0x270a && runeValue <= 0x270b ||
		runeValue == 0x2728 || runeValue == 0x274c || runeValue == 0x274e ||
		runeValue >= 0x2753 && runeValue <= 0x2755 || runeValue == 0x2757 ||
		runeValue >= 0x2795 && runeValue <= 0x2797 ||
		runeValue == 0x27b0 || runeValue == 0x27bf ||
		runeValue >= 0x2b1b && runeValue <= 0x2b1c ||
		runeValue == 0x2b50 || runeValue == 0x2b55
}

// tokenWidthAt resolves tab stops while leaving fixed-width tokens unchanged.
func tokenWidthAt(token displayToken, current int) int {
	if token.dynamicTab {
		return tabWidth - current%tabWidth
	}
	return token.width
}

// truncateLine returns one line unchanged when it already fits.
func truncateLine(line string, width int, tail string) string {
	if VisibleWidth(line) <= width {
		return line
	}

	available := width - VisibleWidth(tail)
	if available < 1 {
		return tail
	}

	var output strings.Builder
	current := 0
	sgrActive := false
	hyperlinkActive := false
	for _, token := range displayTokens(line) {
		if token.escape {
			output.WriteString(token.raw)
			if target, ok := osc8Target(token.raw); ok {
				hyperlinkActive = target != ""
			}
			if resetsAll, hasStyle := sgrProperties(token.raw); resetsAll || hasStyle {
				if resetsAll {
					sgrActive = false
				}
				if hasStyle {
					sgrActive = true
				}
			}
			continue
		}
		tokenWidth := tokenWidthAt(token, current)
		if current+tokenWidth > available {
			break
		}
		output.WriteString(token.raw)
		current += tokenWidth
	}
	if sgrActive {
		output.WriteString(ColorReset)
	}
	if hyperlinkActive {
		output.WriteString("\x1b]8;;\a")
	}
	output.WriteString(tail)
	return output.String()
}

// truncateMiddleLine retains balanced presentation metadata around one unstyled ellipsis.
func truncateMiddleLine(line string, width int) string {
	if VisibleWidth(line) <= width {
		return line
	}
	if width == 1 {
		return "…"
	}

	tokens := displayTokens(line)
	leftBudget := width / 2
	prefixEnd, prefixWidth := truncatePrefixEnd(tokens, leftBudget)
	suffixBudget := width - prefixWidth - 1
	suffixStart := truncateSuffixStart(tokens, prefixEnd, prefixWidth+1, suffixBudget)

	prefix := balanceANSILines([]string{joinDisplayTokens(tokens[:prefixEnd])})[0]
	suffix := ""
	if suffixStart < len(tokens) {
		suffix = ansiContext(tokens[:suffixStart]) + joinDisplayTokens(tokens[suffixStart:])
		suffix = balanceANSILines([]string{suffix})[0]
	}
	return prefix + "…" + suffix
}

// truncatePrefixEnd finds the longest leading token sequence that fits within a cell budget.
func truncatePrefixEnd(tokens []displayToken, width int) (int, int) {
	end := 0
	current := 0
	for index, token := range tokens {
		if token.escape {
			end = index + 1
			continue
		}
		tokenWidth := tokenWidthAt(token, current)
		if tokenWidth > 0 && current+tokenWidth > width {
			break
		}
		current += tokenWidth
		end = index + 1
	}
	if current == 0 {
		return 0, 0
	}
	return end, current
}

// truncateSuffixStart finds the longest trailing token sequence that fits at its final terminal column.
func truncateSuffixStart(tokens []displayToken, minimum, column, width int) int {
	start := len(tokens)
	var additions [tabWidth]int
	for index := len(tokens) - 1; index >= minimum; index-- {
		token := tokens[index]
		if token.escape || token.width == 0 && !token.dynamicTab {
			continue
		}

		var next [tabWidth]int
		for residue := range tabWidth {
			if token.dynamicTab {
				added := tabWidth - residue
				next[residue] = added + additions[0]
				continue
			}
			next[residue] = token.width + additions[(residue+token.width)%tabWidth]
		}
		additions = next
		if additions[column%tabWidth] > width {
			break
		}
		start = index
	}
	return start
}

// ansiContext rebuilds the active presentation state at the end of tokens without visible content.
func ansiContext(tokens []displayToken) string {
	active := make([]string, 0)
	hyperlink := ""
	for _, token := range tokens {
		if !token.escape {
			continue
		}
		if target, ok := osc8Target(token.raw); ok {
			if target == "" {
				hyperlink = ""
			} else {
				hyperlink = token.raw
			}
			continue
		}
		resetsAll, hasStyle := sgrProperties(token.raw)
		if resetsAll {
			active = active[:0]
		}
		if hasStyle {
			active = append(active, token.raw)
		}
	}
	return hyperlink + strings.Join(active, "")
}

// wrapLines performs ANSI-aware word wrapping and preserves explicit blank lines.
func wrapLines(value string, width int) []string {
	tokens := displayTokens(value)
	lines := make([]string, 0, strings.Count(value, "\n")+1)
	line := make([]displayToken, 0)
	lineWidth := 0
	breakPending := false

	flush := func() {
		line = trimTrailingSpaceTokens(line)
		lines = append(lines, joinDisplayTokens(line))
		line = nil
		lineWidth = 0
	}

	for _, token := range tokens {
		if token.newline {
			breakPending = false
			flush()
			continue
		}
		if token.space {
			if breakPending {
				continue
			}
			if lineWidth == 0 {
				continue
			}
			if lineWidth+tokenWidthAt(token, lineWidth) > width {
				breakPending = true
				continue
			}
		}
		if breakPending && !token.escape {
			flush()
			breakPending = false
		}

		tokenWidth := tokenWidthAt(token, lineWidth)
		for tokenWidth > 0 && lineWidth+tokenWidth > width && lineWidth > 0 {
			breakAt := lastSpaceToken(line)
			if breakAt >= 0 {
				carry := trimLeadingSpaceTokens(append([]displayToken(nil), line[breakAt+1:]...))
				line = line[:breakAt]
				flush()
				line = carry
				lineWidth = visibleTokensWidth(line)
				tokenWidth = tokenWidthAt(token, lineWidth)
				continue
			}
			flush()
			tokenWidth = tokenWidthAt(token, lineWidth)
		}

		line = append(line, token)
		lineWidth += tokenWidth
	}
	flush()
	return lines
}

// visibleTokensWidth returns one token line's terminal-cell width.
func visibleTokensWidth(tokens []displayToken) int {
	width := 0
	for _, token := range tokens {
		width += tokenWidthAt(token, width)
	}
	return width
}

// lastSpaceToken returns the final breakable space token in a line.
func lastSpaceToken(tokens []displayToken) int {
	for index := len(tokens) - 1; index >= 0; index-- {
		if tokens[index].space {
			return index
		}
	}
	return -1
}

// trimLeadingSpaceTokens removes layout spaces while retaining leading escape sequences.
func trimLeadingSpaceTokens(tokens []displayToken) []displayToken {
	prefix := make([]displayToken, 0)
	index := 0
	for index < len(tokens) && (tokens[index].space || tokens[index].escape) {
		if tokens[index].escape {
			prefix = append(prefix, tokens[index])
		}
		index++
	}
	return append(prefix, tokens[index:]...)
}

// trimTrailingSpaceTokens removes layout spaces while retaining trailing control sequences.
func trimTrailingSpaceTokens(tokens []displayToken) []displayToken {
	index := len(tokens)
	hasSpace := false
	for index > 0 && (tokens[index-1].space || tokens[index-1].escape) {
		hasSpace = hasSpace || tokens[index-1].space
		index--
	}
	if !hasSpace {
		return tokens
	}

	trimmed := append([]displayToken(nil), tokens[:index]...)
	for _, token := range tokens[index:] {
		if token.escape {
			trimmed = append(trimmed, token)
		}
	}
	return trimmed
}

// joinDisplayTokens rebuilds the exact text represented by tokens.
func joinDisplayTokens(tokens []displayToken) string {
	var output strings.Builder
	for _, token := range tokens {
		output.WriteString(token.raw)
	}
	return output.String()
}

// balanceANSILines closes and reopens SGR styles so inserted line breaks cannot color layout padding or borders.
func balanceANSILines(lines []string) []string {
	balanced := make([]string, len(lines))
	active := make([]string, 0)
	activeHyperlink := ""
	for index, line := range lines {
		prefix := activeHyperlink + strings.Join(active, "")
		for _, token := range displayTokens(line) {
			if !token.escape {
				continue
			}
			if target, ok := osc8Target(token.raw); ok {
				if target == "" {
					activeHyperlink = ""
				} else {
					activeHyperlink = token.raw
				}
				continue
			}
			if !isSGRSequence(token.raw) {
				continue
			}
			resetsAll, hasStyle := sgrProperties(token.raw)
			if resetsAll {
				active = active[:0]
			}
			if hasStyle {
				active = append(active, token.raw)
			}
		}

		balanced[index] = prefix + line
		if len(active) > 0 {
			balanced[index] += ColorReset
		}
		if activeHyperlink != "" {
			balanced[index] += "\x1b]8;;\a"
		}
	}
	return balanced
}

// isSGRSequence reports whether sequence changes Select Graphic Rendition state.
func isSGRSequence(sequence string) bool {
	return len(sequence) >= 3 && strings.HasPrefix(sequence, "\x1b[") && strings.HasSuffix(sequence, "m")
}

// sgrProperties reports whether an SGR sequence resets all state and whether it also applies nonzero styling.
func sgrProperties(sequence string) (bool, bool) {
	if !isSGRSequence(sequence) {
		return false, false
	}
	parameters := sequence[2 : len(sequence)-1]
	parts := strings.Split(parameters, ";")
	resetsAll := parameters == ""
	hasStyle := false
	for index := 0; index < len(parts); index++ {
		part := parts[index]
		if part == "" || part == "0" {
			resetsAll = true
			continue
		}
		hasStyle = true
		if part != "38" && part != "48" && part != "58" || index+1 >= len(parts) {
			continue
		}

		// Extended-color payload zeros are channel or palette values, not reset parameters.
		switch parts[index+1] {
		case "5":
			index = min(index+2, len(parts)-1)
		case "2":
			payloadEnd := index + 4
			if index+2 < len(parts) && parts[index+2] == "" {
				payloadEnd++
			}
			index = min(payloadEnd, len(parts)-1)
		}
	}
	return resetsAll, hasStyle
}

// osc8Target returns an OSC 8 hyperlink target, including an empty target for a closing sequence.
func osc8Target(sequence string) (string, bool) {
	if !strings.HasPrefix(sequence, "\x1b]8;") {
		return "", false
	}
	content := strings.TrimPrefix(sequence, "\x1b]8;")
	content = strings.TrimSuffix(content, "\a")
	content = strings.TrimSuffix(content, "\x1b\\")
	separator := strings.IndexByte(content, ';')
	if separator < 0 {
		return "", false
	}
	return content[separator+1:], true
}
