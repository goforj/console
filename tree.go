package console

import "strings"

// TreeNode contains one label and its ordered child nodes.
// @group Trees
type TreeNode struct {
	// Label is displayed on the node's first physical line.
	Label string
	// Children are displayed in their supplied order beneath the node.
	Children []TreeNode
}

// Node creates a tree node and preserves the supplied child order.
// @group Trees
func Node(label string, children ...TreeNode) TreeNode {
	return TreeNode{Label: label, Children: append([]TreeNode(nil), children...)}
}

// Tree prints a static tree followed by a newline.
// Empty trees produce no output.
// @group Trees
func (c *Console) Tree(nodes ...TreeNode) {
	c.printLayout(c.RenderTree(nodes...))
}

// RenderTree returns a static tree without a trailing newline.
// Root labels remain unprefixed, while descendants use connectors selected by the console's Unicode policy.
// @group Trees
func (c *Console) RenderTree(nodes ...TreeNode) string {
	if len(nodes) == 0 {
		return ""
	}

	characters := c.treeCharacters()
	lines := make([]string, 0, len(nodes))
	for _, node := range nodes {
		c.renderTreeNode(node, "", true, true, characters, &lines)
	}
	return strings.Join(lines, "\n")
}

// Tree prints a static tree through the default console.
// @group Trees
func Tree(nodes ...TreeNode) { Default().Tree(nodes...) }

// RenderTree renders a static tree through the default console.
// @group Trees
func RenderTree(nodes ...TreeNode) string { return Default().RenderTree(nodes...) }

// treeCharacters contains the equal-width connector units used to preserve hanging indentation.
type treeCharacters struct {
	branch   string
	last     string
	vertical string
	indent   string
}

// treeCharacters returns the connector vocabulary selected by the console's Unicode policy.
func (c *Console) treeCharacters() treeCharacters {
	if !c.unicodeEnabled {
		return treeCharacters{
			branch:   "|-- ",
			last:     "`-- ",
			vertical: "|   ",
			indent:   "    ",
		}
	}
	return treeCharacters{
		branch:   "├── ",
		last:     "└── ",
		vertical: "│   ",
		indent:   "    ",
	}
}

// renderTreeNode appends one node and its descendants while carrying visible ancestry connectors forward.
func (c *Console) renderTreeNode(
	node TreeNode,
	ancestry string,
	root bool,
	last bool,
	characters treeCharacters,
	lines *[]string,
) {
	firstPrefix := ""
	continuationPrefix := ""
	childAncestry := ancestry
	if !root {
		branch := characters.branch
		continuation := characters.vertical
		if last {
			branch = characters.last
			continuation = characters.indent
		}
		firstPrefix = ancestry + branch
		continuationPrefix = ancestry + continuation
		childAncestry = continuationPrefix
	}

	*lines = append(*lines, c.renderTreeLabel(node.Label, firstPrefix, continuationPrefix)...)
	for index, child := range node.Children {
		c.renderTreeNode(child, childAncestry, false, index == len(node.Children)-1, characters, lines)
	}
}

// renderTreeLabel sanitizes and wraps one label without allowing it to disturb connector geometry.
func (c *Console) renderTreeLabel(label, firstPrefix, continuationPrefix string) []string {
	label = singleLineLayoutText(label)
	contentWidth := max(c.Width()-VisibleWidth(firstPrefix), 1)
	wrapped := strings.Split(Wrap(label, contentWidth), "\n")
	lines := make([]string, 0, len(wrapped))
	for index, line := range wrapped {
		prefix := continuationPrefix
		if index == 0 {
			prefix = firstPrefix
		}
		lines = append(lines, c.styleTreePrefix(prefix)+c.truncate(line, contentWidth))
	}
	return lines
}

// styleTreePrefix keeps labels in the caller's style while muting visible connector prefixes.
func (c *Console) styleTreePrefix(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		return prefix
	}
	return c.Colorize(ColorGray, prefix)
}
