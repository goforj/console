package console

import (
	"sync"
	"time"
)

// Loader presents one transient activity line on terminals and stable semantic lines in redirected output.
// A Loader is concurrency-safe, single-use, and must be constructed with Console.Loader or NewLoader;
// the first call to Stop, Success, Warn, or Fail wins.
//
// Example: run a loader to completion
//
//	animations := false
//	color := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		ColorEnabled:      &color,
//		UnicodeEnabled:    &unicode,
//	}))
//	var loader *console.Loader = console.NewLoader("Building application")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Building application
//	loader.Success("Application ready")
//	// ✔ Application ready
type Loader struct {
	console *Console

	mu      sync.Mutex
	message string
	state   loaderState
	dynamic bool
	frame   int
	stop    chan struct{}
	done    chan struct{}
}

// loaderState identifies the one-way loader lifecycle.
type loaderState uint8

const (
	loaderReady loaderState = iota
	loaderRunning
	loaderFinished
)

// loaderTicker abstracts ticker timing so lifecycle tests do not depend on wall-clock sleeps.
type loaderTicker interface {
	Ticks() <-chan time.Time
	Stop()
}

// realLoaderTicker adapts time.Ticker to loaderTicker.
type realLoaderTicker struct {
	ticker *time.Ticker
}

// Ticks returns the underlying ticker channel.
func (t *realLoaderTicker) Ticks() <-chan time.Time {
	return t.ticker.C
}

// Stop releases the underlying ticker resources.
func (t *realLoaderTicker) Stop() {
	t.ticker.Stop()
}

// newRealLoaderTicker creates the production animation ticker.
func newRealLoaderTicker(interval time.Duration) loaderTicker {
	return &realLoaderTicker{ticker: time.NewTicker(interval)}
}

// Loader constructs a loader without starting it.
func (c *Console) Loader(message string) *Loader {
	return &Loader{console: c, message: normalizeTransientMessage(message)}
}

// NewLoader constructs a loader using a snapshot of the current default console.
// It does not start the loader.
//
// Example:
//
//	animations := false
//	color := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		ColorEnabled:      &color,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Downloading modules")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Downloading modules
//	loader.Success("Modules ready")
//	// ✔ Modules ready
func NewLoader(message string) *Loader {
	return Default().Loader(message)
}

// Start begins the loader and is harmless when called more than once.
// Animated loaders can return ErrTransientActive when another live display owns the same console.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Building application")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Building application
//	loader.Stop()
func (l *Loader) Start() error {
	l.mu.Lock()
	if l.state != loaderReady {
		l.mu.Unlock()
		return nil
	}

	l.dynamic = l.console.shouldAnimate() && len(l.console.marks.SpinnerFrames) > 0
	if l.dynamic {
		if err := l.console.acquireTransient(l); err != nil {
			l.mu.Unlock()
			return err
		}
		l.stop = make(chan struct{})
		l.done = make(chan struct{})
		l.state = loaderRunning
		go l.animate(l.stop, l.done)
		l.mu.Unlock()
		l.console.renderTransient(l)
		return nil
	}

	l.state = loaderRunning
	message := l.message
	l.console.Action(message)
	l.mu.Unlock()
	return nil
}

// Update changes the loader message and immediately redraws an active animation.
// Updates after a terminal operation are ignored.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Downloading modules")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Downloading modules
//	loader.Update("Verifying modules")
//	loader.Success("")
//	// ✔ Verifying modules
func (l *Loader) Update(message string) {
	l.mu.Lock()
	if l.state == loaderFinished {
		l.mu.Unlock()
		return
	}
	l.message = normalizeTransientMessage(message)
	dynamic := l.state == loaderRunning && l.dynamic
	l.mu.Unlock()
	if dynamic {
		l.console.renderTransient(l)
	}
}

// Stop removes the transient loader without printing a completion message.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Checking configuration")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Checking configuration
//	loader.Stop()
func (l *Loader) Stop() {
	l.finish(loaderFinishStop, "")
}

// Success completes the loader with a success message.
// An empty message reuses the loader's current message.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Publishing release")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Publishing release
//	loader.Success("Release published")
//	// ✔ Release published
func (l *Loader) Success(message string) {
	l.finish(loaderFinishSuccess, message)
}

// Warn completes the loader with a warning message.
// An empty message reuses the loader's current message.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Checking optional tools")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Checking optional tools
//	loader.Warn("Optional tool not found")
//	// ! Optional tool not found
func (l *Loader) Warn(message string) {
	l.finish(loaderFinishWarn, message)
}

// Fail completes the loader with an error message on stderr.
// An empty message reuses the loader's current message.
//
// Example:
//
//	animations := false
//	unicode := true
//	console.SetDefault(console.New(console.Config{
//		AnimationsEnabled: &animations,
//		UnicodeEnabled:    &unicode,
//	}))
//	loader := console.NewLoader("Uploading release")
//	if err := loader.Start(); err != nil {
//		panic(err)
//	}
//	// · Uploading release
//	loader.Fail("Registry refused upload")
//	// ✖ Registry refused upload
func (l *Loader) Fail(message string) {
	l.finish(loaderFinishFail, message)
}

// loaderFinish identifies one terminal loader outcome.
type loaderFinish uint8

const (
	loaderFinishStop loaderFinish = iota
	loaderFinishSuccess
	loaderFinishWarn
	loaderFinishFail
)

// finish performs the winning terminal transition and waits for animation shutdown before durable output.
func (l *Loader) finish(outcome loaderFinish, message string) {
	l.mu.Lock()
	if l.state == loaderFinished {
		l.mu.Unlock()
		return
	}
	wasRunning := l.state == loaderRunning
	dynamic := wasRunning && l.dynamic
	stop := l.stop
	done := l.done
	if message == "" {
		message = l.message
	} else {
		message = normalizeTransientMessage(message)
	}
	l.state = loaderFinished
	l.mu.Unlock()

	if dynamic {
		close(stop)
		<-done
		l.console.releaseTransient(l, outcome != loaderFinishStop)
	}

	switch outcome {
	case loaderFinishSuccess:
		l.console.Success(message)
	case loaderFinishWarn:
		l.console.Warn(message)
	case loaderFinishFail:
		l.console.Error(message)
	}
}

// animate advances frames until the winning terminal operation closes stop.
func (l *Loader) animate(stop <-chan struct{}, done chan<- struct{}) {
	ticker := l.console.newTicker(l.console.loaderInterval)
	defer close(done)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.Ticks():
			l.mu.Lock()
			if l.state == loaderRunning {
				l.frame++
			}
			running := l.state == loaderRunning
			l.mu.Unlock()
			if running {
				l.console.renderTransient(l)
			}
		case <-stop:
			return
		}
	}
}

// renderTransient snapshots a complete carriage-return frame while the console owns output coordination.
func (l *Loader) renderTransient() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.state != loaderRunning || !l.dynamic || len(l.console.marks.SpinnerFrames) == 0 {
		return ""
	}
	frames := l.console.marks.SpinnerFrames
	width := max(l.console.Width(), 1)
	frame := l.console.truncate(singleLineLayoutText(frames[l.frame%len(frames)]), width)
	frameWidth := VisibleWidth(frame)
	if frameWidth == 0 {
		return clearTransientLine
	}
	value := l.console.Colorize(ColorGreen, frame)
	messageWidth := width - frameWidth - 1
	if messageWidth < 1 {
		return clearTransientLine + value
	}
	message := l.console.truncate(l.message, messageWidth)
	if message == "" {
		return clearTransientLine + value
	}
	return clearTransientLine + value + " " + message
}
