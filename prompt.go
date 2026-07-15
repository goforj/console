package console

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrNonInteractive is returned when a prompt would read from a console that is not interactive.
// Set Config.InteractiveEnabled when intentionally driving prompts with an injected reader.
// @group Prompts
var ErrNonInteractive = errors.New("console: prompting requires an interactive console")

// Ask prompts for one trimmed line of input.
// An empty line is returned as an empty string; EOF without input is returned as an error.
// @group Prompts
func (c *Console) Ask(prompt string) (string, error) {
	return c.ask(prompt, nil)
}

// AskDefault prompts for one trimmed line and returns defaultValue when the line is empty.
// @group Prompts
func (c *Console) AskDefault(prompt, defaultValue string) (string, error) {
	return c.ask(prompt, &defaultValue)
}

// Confirm prompts until it reads yes or no, using defaultValue for an empty line.
// Accepted answers are y, yes, n, and no in any letter case.
// @group Prompts
func (c *Console) Confirm(prompt string, defaultValue bool) (bool, error) {
	if !c.IsInteractive() {
		return false, ErrNonInteractive
	}

	c.inputMu.Lock()
	defer c.inputMu.Unlock()

	hint := "[y/N]"
	if defaultValue {
		hint = "[Y/n]"
	}
	for {
		answer, err := c.promptLine(prompt + " " + hint)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return defaultValue, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			c.Warn("Please answer yes or no.")
		}
	}
}

// Choose prints numbered options and returns the selected value.
// defaultIndex is zero-based; use -1 to require an explicit choice.
// @group Prompts
func (c *Console) Choose(prompt string, options []string, defaultIndex int) (string, error) {
	if len(options) == 0 {
		return "", errors.New("console: choose requires at least one option")
	}
	if defaultIndex < -1 || defaultIndex >= len(options) {
		return "", fmt.Errorf("console: default choice index %d is outside 0..%d", defaultIndex, len(options)-1)
	}
	if !c.IsInteractive() {
		return "", ErrNonInteractive
	}

	c.inputMu.Lock()
	defer c.inputMu.Unlock()

	c.write(c.stdout, singleLineLayoutText(prompt)+"\n"+c.renderList(options, true), true)
	for {
		label := fmt.Sprintf("Choose [1-%d]", len(options))
		if defaultIndex >= 0 {
			label = fmt.Sprintf("Choose [1-%d, default %d]", len(options), defaultIndex+1)
		}
		answer, err := c.promptLine(label)
		if err != nil {
			return "", err
		}
		if answer == "" && defaultIndex >= 0 {
			return options[defaultIndex], nil
		}
		selection, err := strconv.Atoi(answer)
		if err == nil && selection >= 1 && selection <= len(options) {
			return options[selection-1], nil
		}
		c.Warn(fmt.Sprintf("Choose a number from 1 to %d.", len(options)))
	}
}

// Ask prompts through the default console.
// @group Prompts
func Ask(prompt string) (string, error) { return Default().Ask(prompt) }

// AskDefault prompts with a default through the default console.
// @group Prompts
func AskDefault(prompt, defaultValue string) (string, error) {
	return Default().AskDefault(prompt, defaultValue)
}

// Confirm asks for confirmation through the default console.
// @group Prompts
func Confirm(prompt string, defaultValue bool) (bool, error) {
	return Default().Confirm(prompt, defaultValue)
}

// Choose asks the user to select an option through the default console.
// @group Prompts
func Choose(prompt string, options []string, defaultIndex int) (string, error) {
	return Default().Choose(prompt, options, defaultIndex)
}

// ask serializes a complete prompt session so buffered input cannot be consumed by another caller.
func (c *Console) ask(prompt string, defaultValue *string) (string, error) {
	if !c.IsInteractive() {
		return "", ErrNonInteractive
	}

	c.inputMu.Lock()
	defer c.inputMu.Unlock()

	label := prompt
	if defaultValue != nil {
		label += " [" + *defaultValue + "]"
	}
	answer, err := c.promptLine(label)
	if err != nil {
		return "", err
	}
	if answer == "" && defaultValue != nil {
		return *defaultValue, nil
	}
	return answer, nil
}

// promptLine owns the output session until input returns so messages cannot erase a live prompt.
func (c *Console) promptLine(prompt string) (string, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	c.transientMu.Lock()
	c.promptActive = true
	c.transientMu.Unlock()
	c.writePrompt(prompt)
	answer, lineTerminated, err := c.readPromptLine()
	c.resumeTransient(lineTerminated)
	return answer, err
}

// writePrompt renders one prompt without a newline because interactive terminals echo submitted input.
func (c *Console) writePrompt(prompt string) {
	pointer := c.Colorize(ColorCyan, singleLineLayoutText(c.marks.Pointer))
	c.writeCoordinated(c.stdout, pointer+" "+singleLineLayoutText(prompt)+": ", true)
}

// readPromptLine accepts a final unterminated value while preserving EOF as distinct from an empty line.
func (c *Console) readPromptLine() (string, bool, error) {
	line, err := c.stdin.ReadString('\n')
	line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))
	if err == nil {
		return line, true, nil
	}
	if errors.Is(err, io.EOF) && line != "" {
		return line, false, nil
	}
	return "", false, err
}
