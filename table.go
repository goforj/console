package console

import "strings"

// TableOption configures one rendered table.
type TableOption func(*tableOptions)

// TableCompact removes the outer frame and separates columns with two spaces.
// A compact table with headers retains one horizontal separator for readability.
func TableCompact() TableOption {
	return func(options *tableOptions) {
		options.compact = true
	}
}

// TableWidths sets content widths by zero-based column position.
// Values less than one leave that column automatic, and configured widths may
// still shrink when the complete table would exceed the console width.
func TableWidths(widths ...int) TableOption {
	configured := append([]int(nil), widths...)
	return func(options *tableOptions) {
		options.widths = append([]int(nil), configured...)
	}
}

// TableRightAlign right-aligns the headers and values in the selected zero-based columns.
// Negative and out-of-range columns are ignored.
func TableRightAlign(columns ...int) TableOption {
	configured := append([]int(nil), columns...)
	return func(options *tableOptions) {
		for _, column := range configured {
			if column >= 0 {
				options.alignments[column] = tableAlignRight
			}
		}
	}
}

// TableCenterAlign centers the headers and values in the selected zero-based columns.
// Negative and out-of-range columns are ignored.
func TableCenterAlign(columns ...int) TableOption {
	configured := append([]int(nil), columns...)
	return func(options *tableOptions) {
		for _, column := range configured {
			if column >= 0 {
				options.alignments[column] = tableAlignCenter
			}
		}
	}
}

// Table prints a table followed by a newline.
// The default presentation is bordered; ragged rows are padded, and cells wrap
// when their automatic or configured columns must shrink.
func (c *Console) Table(headers []string, rows [][]string, options ...TableOption) {
	rendered := c.RenderTable(headers, rows, options...)
	if rendered == "" {
		return
	}
	c.write(c.stdout, rendered+"\n", true)
}

// RenderTable returns a table without a trailing newline.
// Empty headers and rows produce an empty string.
func (c *Console) RenderTable(headers []string, rows [][]string, options ...TableOption) string {
	columnCount := len(headers)
	for _, row := range rows {
		columnCount = max(columnCount, len(row))
	}
	if columnCount == 0 {
		return ""
	}

	configuration := tableOptions{alignments: make(map[int]tableAlignment)}
	for _, option := range options {
		if option != nil {
			option(&configuration)
		}
	}

	widths := make([]int, columnCount)
	for index := range widths {
		widths[index] = 1
	}
	measure := func(cells []string) {
		for column, cell := range cells {
			cell = ExpandTabs(sanitizeLayoutText(cell, true))
			for _, line := range strings.Split(cell, "\n") {
				widths[column] = max(widths[column], VisibleWidth(line))
			}
		}
	}
	measure(headers)
	for _, row := range rows {
		measure(row)
	}
	for column, width := range configuration.widths {
		if column >= len(widths) {
			break
		}
		if width > 0 {
			widths[column] = min(width, max(c.Width(), 1))
		}
	}
	c.fitTableWidths(widths, configuration.compact)

	borders := c.borders()
	if configuration.compact {
		return c.renderCompactTable(headers, rows, widths, borders, configuration.alignments)
	}
	return c.renderBorderedTable(headers, rows, widths, borders, configuration.alignments)
}

// Table prints a table through the default console.
func Table(headers []string, rows [][]string, options ...TableOption) {
	Default().Table(headers, rows, options...)
}

// RenderTable renders a table using the default console.
func RenderTable(headers []string, rows [][]string, options ...TableOption) string {
	return Default().RenderTable(headers, rows, options...)
}

// tableOptions contains normalized functional option state.
type tableOptions struct {
	compact    bool
	widths     []int
	alignments map[int]tableAlignment
}

// tableAlignment identifies the deliberately small set of cell alignment policies.
type tableAlignment uint8

const (
	tableAlignLeft tableAlignment = iota
	tableAlignRight
	tableAlignCenter
)

// renderBorderedTable assembles the default framed presentation.
func (c *Console) renderBorderedTable(
	headers []string,
	rows [][]string,
	widths []int,
	borders borderCharacters,
	alignments map[int]tableAlignment,
) string {
	borderLine := func(left, join, right string) string {
		var line strings.Builder
		line.WriteString(left)
		for column, width := range widths {
			if column > 0 {
				line.WriteString(join)
			}
			line.WriteString(strings.Repeat(borders.horizontal, width+2))
		}
		line.WriteString(right)
		return c.Colorize(ColorGray, line.String())
	}

	lines := []string{borderLine(borders.topLeft, borders.topJoin, borders.topRight)}
	if len(headers) > 0 {
		lines = append(lines, c.renderTableRow(headers, widths, borders, alignments, true, false)...)
		if len(rows) > 0 {
			lines = append(lines, borderLine(borders.middleLeft, borders.middleJoin, borders.middleRight))
		}
	}
	for _, row := range rows {
		lines = append(lines, c.renderTableRow(row, widths, borders, alignments, false, false)...)
	}
	lines = append(lines, borderLine(borders.bottomLeft, borders.bottomJoin, borders.bottomRight))
	return strings.Join(lines, "\n")
}

// renderCompactTable assembles aligned columns without an outer or vertical frame.
func (c *Console) renderCompactTable(
	headers []string,
	rows [][]string,
	widths []int,
	borders borderCharacters,
	alignments map[int]tableAlignment,
) string {
	lines := make([]string, 0, len(rows)+2)
	if len(headers) > 0 {
		lines = append(lines, c.renderTableRow(headers, widths, borders, alignments, true, true)...)
		segments := make([]string, len(widths))
		for column, width := range widths {
			segments[column] = strings.Repeat(borders.horizontal, width)
		}
		lines = append(lines, c.Colorize(ColorGray, strings.Join(segments, "  ")))
	}
	for _, row := range rows {
		lines = append(lines, c.renderTableRow(row, widths, borders, alignments, false, true)...)
	}
	return strings.Join(lines, "\n")
}

// fitTableWidths proportionally shrinks wide columns until the selected presentation fits when possible.
func (c *Console) fitTableWidths(widths []int, compact bool) {
	total := tableWidth(widths, compact)
	excess := total - c.Width()
	for excess > 0 {
		shrinkableColumns := 0
		for _, width := range widths {
			if width > 1 {
				shrinkableColumns++
			}
		}
		if shrinkableColumns == 0 {
			return
		}

		step := max(excess/shrinkableColumns, 1)
		changed := 0
		for index, width := range widths {
			if width <= 1 || excess == 0 {
				continue
			}
			reduction := min(width-1, min(step, excess))
			widths[index] -= reduction
			excess -= reduction
			changed += reduction
		}
		if changed == 0 {
			return
		}
	}
}

// tableWidth returns the total visible width including the selected separators and borders.
func tableWidth(widths []int, compact bool) int {
	if len(widths) == 0 {
		return 0
	}
	if compact {
		total := (len(widths) - 1) * 2
		for _, width := range widths {
			total += width
		}
		return total
	}

	total := 1
	for _, width := range widths {
		total += width + 3
	}
	return total
}

// renderTableRow normalizes ragged, multiline, and wrapped cells into equal-height physical rows.
func (c *Console) renderTableRow(
	cells []string,
	widths []int,
	borders borderCharacters,
	alignments map[int]tableAlignment,
	header bool,
	compact bool,
) []string {
	cellLines := make([][]string, len(widths))
	height := 1
	for column, width := range widths {
		cell := ""
		if column < len(cells) {
			cell = ExpandTabs(sanitizeLayoutText(cells[column], true))
		}
		cellLines[column] = strings.Split(Wrap(cell, width), "\n")
		for index, line := range cellLines[column] {
			cellLines[column][index] = c.truncate(line, width)
		}
		height = max(height, len(cellLines[column]))
	}

	lines := make([]string, 0, height)
	for rowLine := 0; rowLine < height; rowLine++ {
		var line strings.Builder
		if !compact {
			line.WriteString(c.Colorize(ColorGray, borders.vertical))
		}
		for column, width := range widths {
			if column > 0 {
				if compact {
					line.WriteString("  ")
				} else {
					line.WriteString(c.Colorize(ColorGray, borders.vertical))
				}
			}
			value := ""
			if rowLine < len(cellLines[column]) {
				value = cellLines[column][rowLine]
			}
			value = alignTableValue(value, width, alignments[column])
			if compact && column == len(widths)-1 {
				value = strings.TrimRight(value, " ")
			}
			if header {
				value = c.Style(value, StyleBold)
			}
			if compact {
				line.WriteString(value)
			} else {
				line.WriteByte(' ')
				line.WriteString(value)
				line.WriteByte(' ')
			}
		}
		if !compact {
			line.WriteString(c.Colorize(ColorGray, borders.vertical))
		} else {
			trimmed := strings.TrimRight(line.String(), " ")
			line.Reset()
			line.WriteString(trimmed)
		}
		lines = append(lines, line.String())
	}
	return lines
}

// alignTableValue pads one ANSI-aware cell according to its column policy.
func alignTableValue(value string, width int, alignment tableAlignment) string {
	padding := max(width-VisibleWidth(value), 0)
	left := 0
	right := padding
	switch alignment {
	case tableAlignRight:
		left = padding
		right = 0
	case tableAlignCenter:
		left = padding / 2
		right = padding - left
	}
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
}
