// Package console provides lightweight building blocks for polished command-line output.
//
// It includes semantic messages, ANSI-aware text utilities, composable layouts,
// tables, boxes, trees, prompts, loaders, and progress without imposing a
// full-screen event loop.
// Package-level helpers use Default; applications that need isolated writers or
// deterministic behavior can construct a Console with New.
//
//go:generate go -C docs run ./readme
package console
