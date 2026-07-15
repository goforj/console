package console

import "strings"

// BoxOption configures one rendered box.
// @group Boxes
type BoxOption func(*boxOptions)

// BoxTitle adds a title to the top border.
// @group Boxes
func BoxTitle(title string) BoxOption {
	return func(options *boxOptions) {
		options.title = title
	}
}

// BoxWidth fixes the total visible width, including borders and padding.
// Values below the structural minimum expand enough to preserve a valid frame.
// Values less than one select an automatic width bounded by the console width.
// @group Boxes
func BoxWidth(width int) BoxOption {
	return func(options *boxOptions) {
		options.width = width
	}
}

// BoxPadding sets the horizontal padding on both sides of the content.
// Negative values are treated as zero.
// @group Boxes
func BoxPadding(padding int) BoxOption {
	return func(options *boxOptions) {
		options.padding = max(padding, 0)
	}
}

// BoxColor sets the ANSI color used for borders when styling is enabled.
// An empty color leaves borders unstyled.
// @group Boxes
func BoxColor(color string) BoxOption {
	return func(options *boxOptions) {
		options.color = color
	}
}

// Box prints content inside a box followed by a newline.
// @group Boxes
func (c *Console) Box(content string, options ...BoxOption) {
	c.write(c.stdout, c.RenderBox(content, options...)+"\n", true)
}

// RenderBox returns content inside a box without a trailing newline.
// @group Boxes
func (c *Console) RenderBox(content string, options ...BoxOption) string {
	configuration := boxOptions{padding: 1, color: ColorGray}
	for _, option := range options {
		if option != nil {
			option(&configuration)
		}
	}

	borders := c.borders()
	title := singleLineLayoutText(configuration.title)
	content = ExpandTabs(sanitizeLayoutText(content, true))
	contentLines := strings.Split(content, "\n")
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	minimumWidth := 2 + configuration.padding*2 + 1
	outerWidth := configuration.width
	if outerWidth < 1 {
		outerWidth = minimumWidth
		for _, line := range contentLines {
			outerWidth = max(outerWidth, VisibleWidth(line)+configuration.padding*2+2)
		}
		if title != "" {
			outerWidth = max(outerWidth, VisibleWidth(title)+5)
		}
		outerWidth = min(outerWidth, max(c.Width(), minimumWidth))
	}
	outerWidth = max(outerWidth, minimumWidth)
	innerWidth := max(outerWidth-2-configuration.padding*2, 1)

	wrappedLines := strings.Split(Wrap(content, innerWidth), "\n")
	wrapped := make([]string, 0, len(wrappedLines))
	for _, wrappedLine := range wrappedLines {
		wrapped = append(wrapped, c.truncate(wrappedLine, innerWidth))
	}
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}

	borderStyle := func(value string) string {
		if configuration.color == "" {
			return value
		}
		return c.Colorize(configuration.color, value)
	}

	lines := make([]string, 0, len(wrapped)+2)
	lines = append(lines, c.renderBoxTop(borders, outerWidth, title, borderStyle))
	for _, line := range wrapped {
		body := strings.Repeat(" ", configuration.padding) + PadRight(line, innerWidth)
		body += strings.Repeat(" ", configuration.padding)
		lines = append(lines, borderStyle(borders.vertical)+body+borderStyle(borders.vertical))
	}
	lines = append(lines, borderStyle(borders.bottomLeft)+borderStyle(strings.Repeat(borders.horizontal, outerWidth-2))+borderStyle(borders.bottomRight))
	return strings.Join(lines, "\n")
}

// Box prints a box through the default console.
// @group Boxes
func Box(content string, options ...BoxOption) { Default().Box(content, options...) }

// RenderBox renders a box using the default console.
// @group Boxes
func RenderBox(content string, options ...BoxOption) string {
	return Default().RenderBox(content, options...)
}

// boxOptions contains normalized functional option state.
type boxOptions struct {
	title   string
	width   int
	padding int
	color   string
}

// borderCharacters contains the line-drawing vocabulary shared by boxes and tables.
type borderCharacters struct {
	horizontal  string
	vertical    string
	topLeft     string
	topJoin     string
	topRight    string
	middleLeft  string
	middleJoin  string
	middleRight string
	bottomLeft  string
	bottomJoin  string
	bottomRight string
}

// borders returns the Unicode or ASCII border set selected by the console.
func (c *Console) borders() borderCharacters {
	if !c.unicodeEnabled {
		return borderCharacters{
			horizontal:  "-",
			vertical:    "|",
			topLeft:     "+",
			topJoin:     "+",
			topRight:    "+",
			middleLeft:  "+",
			middleJoin:  "+",
			middleRight: "+",
			bottomLeft:  "+",
			bottomJoin:  "+",
			bottomRight: "+",
		}
	}
	return borderCharacters{
		horizontal:  "─",
		vertical:    "│",
		topLeft:     "┌",
		topJoin:     "┬",
		topRight:    "┐",
		middleLeft:  "├",
		middleJoin:  "┼",
		middleRight: "┤",
		bottomLeft:  "└",
		bottomJoin:  "┴",
		bottomRight: "┘",
	}
}

// renderBoxTop lays a title into the border without changing the requested width.
func (c *Console) renderBoxTop(
	borders borderCharacters,
	width int,
	title string,
	borderStyle func(string) string,
) string {
	insideWidth := width - 2
	if title == "" || insideWidth < 3 {
		return borderStyle(borders.topLeft) +
			borderStyle(strings.Repeat(borders.horizontal, insideWidth)) +
			borderStyle(borders.topRight)
	}

	label := " " + c.truncate(title, max(insideWidth-3, 0)) + " "
	leftWidth := 1
	rightWidth := max(insideWidth-leftWidth-VisibleWidth(label), 0)
	return borderStyle(borders.topLeft) +
		borderStyle(strings.Repeat(borders.horizontal, leftWidth)) +
		c.Style(label, ColorBoldWhite) +
		borderStyle(strings.Repeat(borders.horizontal, rightWidth)) +
		borderStyle(borders.topRight)
}
