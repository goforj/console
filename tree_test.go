package console

import (
	"strings"
	"testing"
)

// TestNodeConstructsAnOrderedTree verifies the convenience constructor owns its child slice.
func TestNodeConstructsAnOrderedTree(t *testing.T) {
	t.Parallel()

	children := []TreeNode{Node("first"), Node("second")}
	root := Node("root", children...)
	children[0].Label = "changed"

	if root.Label != "root" {
		t.Fatalf("Node().Label = %q, want root", root.Label)
	}
	if got, want := root.Children[0].Label, "first"; got != want {
		t.Fatalf("Node().Children[0].Label = %q, want %q", got, want)
	}
	if got, want := root.Children[1].Label, "second"; got != want {
		t.Fatalf("Node().Children[1].Label = %q, want %q", got, want)
	}
}

// TestRenderTreeUsesCapabilityAppropriateConnectors verifies hierarchy and child order in both display modes.
func TestRenderTreeUsesCapabilityAppropriateConnectors(t *testing.T) {
	t.Parallel()

	tree := Node("project",
		Node("cmd", Node("console")),
		Node("internal"),
		Node("README.md"),
	)
	tests := []struct {
		name           string
		unicodeEnabled bool
		want           string
	}{
		{
			name:           "Unicode",
			unicodeEnabled: true,
			want: "project\n" +
				"├── cmd\n" +
				"│   └── console\n" +
				"├── internal\n" +
				"└── README.md",
		},
		{
			name:           "ASCII",
			unicodeEnabled: false,
			want: "project\n" +
				"|-- cmd\n" +
				"|   `-- console\n" +
				"|-- internal\n" +
				"`-- README.md",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			console, output := newLayoutTestConsole(40, test.unicodeEnabled, false)
			if got := console.RenderTree(tree); got != test.want {
				t.Fatalf("RenderTree() returned:\n%q\nwant:\n%q", got, test.want)
			}
			if got := output.String(); got != "" {
				t.Fatalf("RenderTree() wrote %q, want no output", got)
			}
			console.Tree(tree)
			if got, want := output.String(), test.want+"\n"; got != want {
				t.Fatalf("Tree() wrote:\n%q\nwant:\n%q", got, want)
			}
		})
	}
}

// TestRenderTreeWrapsLabelsWithHangingIndentation verifies continuation rows preserve the tree structure.
func TestRenderTreeWrapsLabelsWithHangingIndentation(t *testing.T) {
	t.Parallel()

	console, _ := newLayoutTestConsole(14, true, false)
	tree := Node("root", Node("alpha beta gamma"), Node("done"))
	want := "root\n" +
		"├── alpha beta\n" +
		"│   gamma\n" +
		"└── done"
	got := console.RenderTree(tree)
	if got != want {
		t.Fatalf("RenderTree() returned %q, want %q", got, want)
	}
	for index, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width > 14 {
			t.Fatalf("line %d width = %d, want at most 14: %q", index, width, line)
		}
	}
}

// TestRenderTreeSanitizesLabels verifies node metadata cannot inject rows or terminal controls.
func TestRenderTreeSanitizesLabels(t *testing.T) {
	t.Parallel()

	console, _ := newLayoutTestConsole(40, true, false)
	tree := Node("root\nnext\x1b[2J\x1b]0;title\a", Node("safe\rchild"))
	if got, want := console.RenderTree(tree), "root next\n└── safe child"; got != want {
		t.Fatalf("RenderTree() returned %q, want %q", got, want)
	}
}

// TestRenderTreeStylesOnlyConnectors verifies labels are not accidentally included in muted connector styling.
func TestRenderTreeStylesOnlyConnectors(t *testing.T) {
	t.Parallel()

	console, _ := newLayoutTestConsole(20, true, true)
	want := "root\n" + ColorGray + "└── " + ColorReset + "child"
	if got := console.RenderTree(Node("root", Node("child"))); got != want {
		t.Fatalf("RenderTree() returned %q, want %q", got, want)
	}
}

// TestEmptyTreesStayEmpty verifies tree helpers do not manufacture a blank output row.
func TestEmptyTreesStayEmpty(t *testing.T) {
	t.Parallel()

	console, output := newLayoutTestConsole(20, true, false)
	if got := console.RenderTree(); got != "" {
		t.Fatalf("RenderTree() returned %q, want empty", got)
	}
	console.Tree()
	if got := output.String(); got != "" {
		t.Fatalf("Tree() wrote %q, want no output", got)
	}
}
