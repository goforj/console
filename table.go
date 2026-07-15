package console

import "strings"

// Table prints a bordered table followed by a newline.
// Ragged rows are padded to the widest row or header, and oversized cells are truncated.
// @group Tables
func (c *Console) Table(headers []string, rows [][]string) {
	rendered := c.RenderTable(headers, rows)
	if rendered == "" {
		return
	}
	c.write(c.stdout, rendered+"\n", true)
}

// RenderTable returns a bordered table without a trailing newline.
// Empty headers and rows produce an empty string.
// @group Tables
func (c *Console) RenderTable(headers []string, rows [][]string) string {
	columnCount := len(headers)
	for _, row := range rows {
		columnCount = max(columnCount, len(row))
	}
	if columnCount == 0 {
		return ""
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
	c.fitTableWidths(widths)

	borders := c.borders()
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
		lines = append(lines, c.renderTableRow(headers, widths, borders, true)...)
		if len(rows) > 0 {
			lines = append(lines, borderLine(borders.middleLeft, borders.middleJoin, borders.middleRight))
		}
	}
	for _, row := range rows {
		lines = append(lines, c.renderTableRow(row, widths, borders, false)...)
	}
	lines = append(lines, borderLine(borders.bottomLeft, borders.bottomJoin, borders.bottomRight))
	return strings.Join(lines, "\n")
}

// Table prints a table through the default console.
// @group Tables
func Table(headers []string, rows [][]string) { Default().Table(headers, rows) }

// RenderTable renders a table using the default console.
// @group Tables
func RenderTable(headers []string, rows [][]string) string {
	return Default().RenderTable(headers, rows)
}

// fitTableWidths proportionally shrinks wide columns until borders fit the console when possible.
func (c *Console) fitTableWidths(widths []int) {
	total := tableWidth(widths)
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

// tableWidth returns the total visible width including padding and borders.
func tableWidth(widths []int) int {
	total := 1
	for _, width := range widths {
		total += width + 3
	}
	return total
}

// renderTableRow normalizes ragged and multiline cells into equal-width physical rows.
func (c *Console) renderTableRow(
	cells []string,
	widths []int,
	borders borderCharacters,
	header bool,
) []string {
	cellLines := make([][]string, len(widths))
	height := 1
	for column := range widths {
		cell := ""
		if column < len(cells) {
			cell = ExpandTabs(sanitizeLayoutText(cells[column], true))
		}
		cellLines[column] = balanceANSILines(strings.Split(cell, "\n"))
		height = max(height, len(cellLines[column]))
	}

	lines := make([]string, 0, height)
	for rowLine := 0; rowLine < height; rowLine++ {
		var line strings.Builder
		line.WriteString(c.Colorize(ColorGray, borders.vertical))
		for column, width := range widths {
			if column > 0 {
				line.WriteString(c.Colorize(ColorGray, borders.vertical))
			}
			value := ""
			if rowLine < len(cellLines[column]) {
				value = c.truncate(cellLines[column][rowLine], width)
			}
			value = PadRight(value, width)
			if header {
				value = c.Style(value, StyleBold)
			}
			line.WriteByte(' ')
			line.WriteString(value)
			line.WriteByte(' ')
		}
		line.WriteString(c.Colorize(ColorGray, borders.vertical))
		lines = append(lines, line.String())
	}
	return lines
}
