package console

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// KeyValue contains one ordered label and value for KeyValues.
// @group Layout
type KeyValue struct {
	// Key is the label displayed in the left column.
	Key string
	// Value is formatted with fmt.Sprint in the right column.
	Value any
}

// KV creates one ordered key/value entry.
// @group Layout
func KV(key string, value any) KeyValue {
	return KeyValue{Key: key, Value: value}
}

// Section prints a visually distinct section heading.
// @group Layout
func (c *Console) Section(title string) {
	title = singleLineLayoutText(title)
	mark := "◇"
	if !c.unicodeEnabled {
		mark = ">"
	}
	c.write(c.stdout, c.Colorize(ColorCyan, mark)+" "+c.Style(title, ColorBoldWhite)+"\n", true)
}

// Rule prints a horizontal rule, optionally interrupted by title.
// @group Layout
func (c *Console) Rule(title string) {
	title = singleLineLayoutText(title)
	character := "─"
	if !c.unicodeEnabled {
		character = "-"
	}
	width := max(c.Width(), 1)
	line := strings.Repeat(character, width)
	if title != "" && width > 4 {
		label := " " + c.truncate(title, width-4) + " "
		prefixWidth := min(2, max(width-VisibleWidth(label), 0))
		remaining := max(width-prefixWidth-VisibleWidth(label), 0)
		line = strings.Repeat(character, prefixWidth) + label + strings.Repeat(character, remaining)
	}
	c.write(c.stdout, c.Colorize(ColorGray, line)+"\n", true)
}

// KeyValues prints ordered and visibly aligned key/value entries.
// @group Layout
func (c *Console) KeyValues(entries ...KeyValue) {
	if len(entries) == 0 {
		return
	}

	keyWidth := 0
	for _, entry := range entries {
		keyWidth = max(keyWidth, VisibleWidth(singleLineLayoutText(entry.Key)))
	}
	availableWidth := max(c.Width(), 1)
	valueFloor := min(3, max(availableWidth-3, 1))
	keyLimit := max(availableWidth-2-valueFloor, 1)
	if keyWidth > keyLimit {
		keyWidth = keyLimit
	}
	valueWidth := max(availableWidth-keyWidth-2, 1)

	var output strings.Builder
	for _, entry := range entries {
		key := c.truncate(singleLineLayoutText(entry.Key), keyWidth)
		value := ExpandTabs(sanitizeLayoutText(fmt.Sprint(entry.Value), true))
		valueLines := strings.Split(Wrap(value, valueWidth), "\n")
		if len(valueLines) == 0 {
			valueLines = []string{""}
		}
		for index, line := range valueLines {
			valueLines[index] = c.truncate(line, valueWidth)
		}
		output.WriteString(c.Colorize(ColorGray, PadRight(key, keyWidth)))
		output.WriteString("  ")
		output.WriteString(valueLines[0])
		output.WriteByte('\n')
		for _, line := range valueLines[1:] {
			output.WriteString(strings.Repeat(" ", keyWidth+2))
			output.WriteString(line)
			output.WriteByte('\n')
		}
	}
	c.write(c.stdout, output.String(), true)
}

// KeyValueMap prints a map in sorted-key order for deterministic output.
// @group Layout
func (c *Console) KeyValueMap(values map[string]any) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]KeyValue, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, KV(key, values[key]))
	}
	c.KeyValues(entries...)
}

// List prints an unordered list and applies hanging indentation to wrapped items.
// @group Layout
func (c *Console) List(items ...string) {
	if len(items) == 0 {
		return
	}
	c.write(c.stdout, c.renderList(items, false), true)
}

// NumberedList prints a one-based ordered list with aligned numeric prefixes.
// @group Layout
func (c *Console) NumberedList(items ...string) {
	if len(items) == 0 {
		return
	}
	c.write(c.stdout, c.renderList(items, true), true)
}

// Section prints a section heading through the default console.
// @group Layout
func Section(title string) { Default().Section(title) }

// Rule prints a horizontal rule through the default console.
// @group Layout
func Rule(title string) { Default().Rule(title) }

// KeyValues prints ordered key/value entries through the default console.
// @group Layout
func KeyValues(entries ...KeyValue) { Default().KeyValues(entries...) }

// KeyValueMap prints a sorted key/value map through the default console.
// @group Layout
func KeyValueMap(values map[string]any) { Default().KeyValueMap(values) }

// List prints an unordered list through the default console.
// @group Layout
func List(items ...string) { Default().List(items...) }

// NumberedList prints an ordered list through the default console.
// @group Layout
func NumberedList(items ...string) { Default().NumberedList(items...) }

// renderList returns one complete list write so concurrent output cannot split its items.
func (c *Console) renderList(items []string, numbered bool) string {
	bullet := singleLineLayoutText(c.marks.Bullet)
	prefixWidth := VisibleWidth(bullet) + 1
	if numbered {
		prefixWidth = len(strconv.Itoa(len(items))) + 2
	}
	contentWidth := max(c.Width()-prefixWidth, 1)

	var output strings.Builder
	for index, item := range items {
		prefix := c.Colorize(ColorGray, bullet) + " "
		if numbered {
			number := strconv.Itoa(index+1) + "."
			prefix = c.Colorize(ColorGray, strings.Repeat(" ", prefixWidth-1-VisibleWidth(number))+number) + " "
		}
		item = ExpandTabs(sanitizeLayoutText(item, true))
		lines := strings.Split(Wrap(item, contentWidth), "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		for lineIndex, line := range lines {
			lines[lineIndex] = c.truncate(line, contentWidth)
		}
		output.WriteString(prefix)
		output.WriteString(lines[0])
		output.WriteByte('\n')
		for _, line := range lines[1:] {
			output.WriteString(strings.Repeat(" ", prefixWidth))
			output.WriteString(line)
			output.WriteByte('\n')
		}
	}
	return output.String()
}
