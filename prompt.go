package console

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ErrNonInteractive is returned when a prompt would read from a console that is not interactive.
// Set Config.InteractiveEnabled when intentionally driving prompts with an injected reader.
//
// Example: recognize a non-interactive prompt
//
//	interactive := false
//	console.SetDefault(console.New(console.Config{InteractiveEnabled: &interactive}))
//	_, err := console.Ask("Name")
//	fmt.Println(errors.Is(err, console.ErrNonInteractive))
//	// true
var ErrNonInteractive = errors.New("console: prompting requires an interactive console")

// Ask prompts for one trimmed line of input.
// An empty line is returned as an empty string; EOF without input is returned as an error.
func (c *Console) Ask(prompt string) (string, error) {
	return c.ask(prompt, nil)
}

// AskDefault prompts for one trimmed line and returns defaultValue when the line is empty.
func (c *Console) AskDefault(prompt, defaultValue string) (string, error) {
	return c.ask(prompt, &defaultValue)
}

// AskSecret prompts for one value without echoing input to the terminal.
// Config.ReadSecret can provide compatible behavior for tests and custom terminals.
func (c *Console) AskSecret(prompt string) (string, error) {
	if !c.IsInteractive() {
		return "", ErrNonInteractive
	}

	c.inputMu.Lock()
	defer c.inputMu.Unlock()

	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	c.transientMu.Lock()
	c.promptActive = true
	c.transientMu.Unlock()
	if err := c.writePrompt(prompt); err != nil {
		return "", errors.Join(err, c.resumeTransient(false))
	}
	secret, err := c.readSecret()
	return secret, errors.Join(err, c.resumeTransient(false))
}

// Confirm prompts until it reads yes or no, using defaultValue for an empty line.
// Accepted answers are y, yes, n, and no in any letter case.
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
			if err := c.writePromptOutput(c.renderSemanticMessage(c.WarnMark(), "Please answer yes or no.")); err != nil {
				return false, err
			}
		}
	}
}

// Choose prints numbered options and returns the selected value.
// defaultIndex is zero-based; use -1 to require an explicit choice.
func (c *Console) Choose(prompt string, options []string, defaultIndex int) (string, error) {
	index, err := c.ChooseIndex(prompt, options, defaultIndex)
	if err != nil {
		return "", err
	}
	return options[index], nil
}

// ChooseIndex prints numbered options and returns the selected zero-based index.
// defaultIndex is zero-based; use -1 to require an explicit choice.
func (c *Console) ChooseIndex(prompt string, options []string, defaultIndex int) (int, error) {
	if len(options) == 0 {
		return -1, errors.New("console: choose requires at least one option")
	}
	if defaultIndex < -1 || defaultIndex >= len(options) {
		return -1, fmt.Errorf("console: default choice index %d is outside 0..%d", defaultIndex, len(options)-1)
	}
	if !c.IsInteractive() {
		return -1, ErrNonInteractive
	}

	c.inputMu.Lock()
	defer c.inputMu.Unlock()

	if err := c.writePromptOutput(singleLineLayoutText(prompt) + "\n" + c.renderList(options, true)); err != nil {
		return -1, err
	}
	for {
		label := fmt.Sprintf("Choose [1-%d]", len(options))
		if defaultIndex >= 0 {
			label = fmt.Sprintf("Choose [1-%d, default %d]", len(options), defaultIndex+1)
		}
		answer, err := c.promptLine(label)
		if err != nil {
			return -1, err
		}
		if answer == "" && defaultIndex >= 0 {
			return defaultIndex, nil
		}
		selection, err := strconv.Atoi(answer)
		if err == nil && selection >= 1 && selection <= len(options) {
			return selection - 1, nil
		}
		warning := fmt.Sprintf("Choose a number from 1 to %d.", len(options))
		if err := c.writePromptOutput(c.renderSemanticMessage(c.WarnMark(), warning)); err != nil {
			return -1, err
		}
	}
}

// Ask prompts through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdin:              strings.NewReader("Ada\n"),
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//	}))
//	name, err := console.Ask("Name")
//	fmt.Printf("%q\n", output.String())
//	// "› Name: "
//	fmt.Println(name, err)
//	// Ada <nil>
func Ask(prompt string) (string, error) { return Default().Ask(prompt) }

// AskDefault prompts with a default through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdin:              strings.NewReader("\n"),
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//	}))
//	environment, err := console.AskDefault("Environment", "production")
//	fmt.Printf("%q\n", output.String())
//	// "› Environment [production]: "
//	fmt.Println(environment, err)
//	// production <nil>
func AskDefault(prompt, defaultValue string) (string, error) {
	return Default().AskDefault(prompt, defaultValue)
}

// AskSecret prompts without echoing input through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//		ReadSecret: func() (string, error) {
//			return "token-value", nil
//		},
//	}))
//	secret, err := console.AskSecret("API token")
//	fmt.Printf("%q\n", output.String())
//	// "› API token: \n"
//	fmt.Println(len(secret), err)
//	// 11 <nil>
func AskSecret(prompt string) (string, error) {
	return Default().AskSecret(prompt)
}

// Confirm asks for confirmation through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdin:              strings.NewReader("yes\n"),
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//	}))
//	confirmed, err := console.Confirm("Deploy now", false)
//	fmt.Printf("%q\n", output.String())
//	// "› Deploy now [y/N]: "
//	fmt.Println(confirmed, err)
//	// true <nil>
func Confirm(prompt string, defaultValue bool) (bool, error) {
	return Default().Confirm(prompt, defaultValue)
}

// Choose asks the user to select an option through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdin:              strings.NewReader("2\n"),
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//	}))
//	channel, err := console.Choose("Release channel", []string{"stable", "beta"}, 0)
//	fmt.Printf("%q\n", output.String())
//	// "Release channel\n1. stable\n2. beta\n› Choose [1-2, default 1]: "
//	fmt.Println(channel, err)
//	// beta <nil>
func Choose(prompt string, options []string, defaultIndex int) (string, error) {
	return Default().Choose(prompt, options, defaultIndex)
}

// ChooseIndex asks the user to select an option index through the default console.
//
// Example:
//
//	var output bytes.Buffer
//	interactive := true
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		Stdin:              strings.NewReader("2\n"),
//		Stdout:             &output,
//		InteractiveEnabled: &interactive,
//		UnicodeEnabled:     &unicode,
//	}))
//	index, err := console.ChooseIndex("Release channel", []string{"stable", "beta"}, 0)
//	fmt.Printf("%q\n", output.String())
//	// "Release channel\n1. stable\n2. beta\n› Choose [1-2, default 1]: "
//	fmt.Println(index, err)
//	// 1 <nil>
func ChooseIndex(prompt string, options []string, defaultIndex int) (int, error) {
	return Default().ChooseIndex(prompt, options, defaultIndex)
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
	if err := c.writePrompt(prompt); err != nil {
		return "", errors.Join(err, c.resumeTransient(false))
	}
	answer, lineTerminated, err := c.readPromptLine()
	return answer, errors.Join(err, c.resumeTransient(lineTerminated))
}

// writePrompt renders one prompt without a newline because interactive terminals echo submitted input.
func (c *Console) writePrompt(prompt string) error {
	pointer := c.Colorize(ColorCyan, singleLineLayoutText(c.marks.Pointer))
	_, err := c.writeCoordinated(c.stdout, pointer+" "+singleLineLayoutText(prompt)+": ", true)
	return err
}

// writePromptOutput reports prelude and retry output failures instead of silently continuing to read input.
func (c *Console) writePromptOutput(value string) error {
	c.sessionMu.RLock()
	_, err := c.writeCoordinated(c.stdout, value, true)
	c.sessionMu.RUnlock()
	if err == nil {
		return nil
	}
	return errors.Join(err, c.resumeTransient(false))
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

// readTerminalSecret reads a password from a terminal-backed input without weakening to echoed input.
func readTerminalSecret(reader io.Reader) (string, error) {
	descriptor, ok := readerDescriptor(reader)
	if !ok {
		return "", errors.New("console: secret input requires a terminal reader or Config.ReadSecret")
	}
	value, err := term.ReadPassword(descriptor)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
