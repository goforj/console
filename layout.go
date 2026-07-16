package console

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// KeyValue contains one ordered label and value for KeyValues.
//
// Example: construct an ordered deployment summary
//
//	entries := []console.KeyValue{
//		{Key: "Mode", Value: "production"},
//		{Key: "Port", Value: 8080},
//	}
//	console.KeyValues(entries...)
//	// Mode  production
//	// Port  8080
type KeyValue struct {
	// Key is the label displayed in the left column.
	Key string
	// Value is formatted with fmt.Sprint in the right column.
	Value any
}

// KV creates one ordered key/value entry.
//
// Example: build an ordered summary entry
//
//	console.KeyValues(console.KV("Region", "eu-west-1"))
//	// Region  eu-west-1
func KV(key string, value any) KeyValue {
	return KeyValue{Key: key, Value: value}
}

// Section prints a visually distinct section heading.
func (c *Console) Section(title string) {
	c.printLayout(c.RenderSection(title))
}

// RenderSection returns a visually distinct section heading without a trailing newline.
func (c *Console) RenderSection(title string) string {
	title = singleLineLayoutText(title)
	mark := "◇"
	if !c.unicodeEnabled {
		mark = ">"
	}
	width := max(c.Width(), 1)
	mark = c.truncate(mark, width)
	remaining := width - VisibleWidth(mark)
	styledMark := c.Colorize(ColorCyan, mark)
	if title == "" || remaining < 2 {
		return styledMark
	}
	title = c.truncate(title, remaining-1)
	return styledMark + " " + c.Style(title, ColorBoldWhite)
}

// Rule prints a horizontal rule, optionally interrupted by title.
func (c *Console) Rule(title string) {
	c.printLayout(c.RenderRule(title))
}

// RenderRule returns a horizontal rule without a trailing newline.
// The optional title interrupts the rule when the configured width allows it.
func (c *Console) RenderRule(title string) string {
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
	return c.Colorize(ColorGray, line)
}

// KeyValues prints ordered and visibly aligned key/value entries.
func (c *Console) KeyValues(entries ...KeyValue) {
	c.printLayout(c.RenderKeyValues(entries...))
}

// RenderKeyValues returns ordered and visibly aligned key/value entries without a trailing newline.
// Empty entries produce an empty string.
func (c *Console) RenderKeyValues(entries ...KeyValue) string {
	if len(entries) == 0 {
		return ""
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
	return strings.TrimSuffix(output.String(), "\n")
}

// KeyValueMap prints a map in sorted-key order for deterministic output.
func (c *Console) KeyValueMap(values map[string]any) {
	c.printLayout(c.RenderKeyValueMap(values))
}

// RenderKeyValueMap returns map entries in sorted-key order without a trailing newline.
// Empty maps produce an empty string.
func (c *Console) RenderKeyValueMap(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]KeyValue, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, KV(key, values[key]))
	}
	return c.RenderKeyValues(entries...)
}

// List prints an unordered list and applies hanging indentation to wrapped items.
func (c *Console) List(items ...string) {
	c.printLayout(c.RenderList(items...))
}

// RenderList returns an unordered list with hanging indentation and no trailing newline.
// Empty items produce an empty string.
func (c *Console) RenderList(items ...string) string {
	return strings.TrimSuffix(c.renderList(items, false), "\n")
}

// NumberedList prints a one-based ordered list with aligned numeric prefixes.
func (c *Console) NumberedList(items ...string) {
	c.printLayout(c.RenderNumberedList(items...))
}

// RenderNumberedList returns a one-based ordered list with hanging indentation and no trailing newline.
// Empty items produce an empty string.
func (c *Console) RenderNumberedList(items ...string) string {
	return strings.TrimSuffix(c.renderList(items, true), "\n")
}

// Section prints a section heading through the default console.
//
// Example: print a deployment section
//
//	console.Section("Deployment")
//	// ◇ Deployment
func Section(title string) { Default().Section(title) }

// RenderSection renders a section heading through the default console.
//
// Example: compose a section heading
//
//	fmt.Println(console.RenderSection("Deployment"))
//	// ◇ Deployment
func RenderSection(title string) string { return Default().RenderSection(title) }

// Rule prints a horizontal rule through the default console.
//
// Example: separate two phases
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	console.SetDefault(console.New(console.Config{Width: 16}))
//	console.Rule("Next")
//	// ── Next ────────
func Rule(title string) { Default().Rule(title) }

// RenderRule renders a horizontal rule through the default console.
//
// Example: compose a phase separator
//
//	previous := console.Default()
//	defer console.SetDefault(previous)
//	console.SetDefault(console.New(console.Config{Width: 16}))
//	fmt.Println(console.RenderRule("Next"))
//	// ── Next ────────
func RenderRule(title string) string { return Default().RenderRule(title) }

// KeyValues prints ordered key/value entries through the default console.
//
// Example: print an ordered deployment summary
//
//	console.KeyValues(
//		console.KV("Mode", "production"),
//		console.KV("Port", 8080),
//	)
//	// Mode  production
//	// Port  8080
func KeyValues(entries ...KeyValue) { Default().KeyValues(entries...) }

// RenderKeyValues renders ordered key/value entries through the default console.
//
// Example: compose an ordered summary
//
//	fmt.Println(console.RenderKeyValues(
//		console.KV("Mode", "production"),
//		console.KV("Port", 8080),
//	))
//	// Mode  production
//	// Port  8080
func RenderKeyValues(entries ...KeyValue) string { return Default().RenderKeyValues(entries...) }

// KeyValueMap prints a sorted key/value map through the default console.
//
// Example: print map values in stable key order
//
//	console.KeyValueMap(map[string]any{"port": 8080, "mode": "production"})
//	// mode  production
//	// port  8080
func KeyValueMap(values map[string]any) { Default().KeyValueMap(values) }

// RenderKeyValueMap renders a sorted key/value map through the default console.
//
// Example: compose deterministic map output
//
//	fmt.Println(console.RenderKeyValueMap(map[string]any{"port": 8080, "mode": "production"}))
//	// mode  production
//	// port  8080
func RenderKeyValueMap(values map[string]any) string { return Default().RenderKeyValueMap(values) }

// List prints an unordered list through the default console.
//
// Example: print a short checklist
//
//	console.List("build", "test", "publish")
//	// • build
//	// • test
//	// • publish
func List(items ...string) { Default().List(items...) }

// RenderList renders an unordered list through the default console.
//
// Example: compose an unordered list
//
//	fmt.Println(console.RenderList("build", "test"))
//	// • build
//	// • test
func RenderList(items ...string) string { return Default().RenderList(items...) }

// NumberedList prints an ordered list through the default console.
//
// Example: print release steps
//
//	console.NumberedList("build", "test", "publish")
//	// 1. build
//	// 2. test
//	// 3. publish
func NumberedList(items ...string) { Default().NumberedList(items...) }

// RenderNumberedList renders an ordered list through the default console.
//
// Example: compose ordered steps
//
//	fmt.Println(console.RenderNumberedList("build", "test"))
//	// 1. build
//	// 2. test
func RenderNumberedList(items ...string) string { return Default().RenderNumberedList(items...) }

// printLayout writes one rendered layout value followed by exactly one newline.
func (c *Console) printLayout(rendered string) {
	if rendered == "" {
		return
	}
	c.write(c.stdout, rendered+"\n", true)
}

// renderList returns one complete list write so concurrent output cannot split its items.
func (c *Console) renderList(items []string, numbered bool) string {
	if len(items) == 0 {
		return ""
	}

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
