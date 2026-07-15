package console

import (
	"sync"
	"time"
)

// Loader presents one transient activity line on terminals and stable semantic lines in redirected output.
// A Loader is concurrency-safe, single-use, and must be constructed with Console.Loader or NewLoader;
// the first call to Stop, Success, Warn, or Fail wins.
// @group Loaders
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
// @group Loaders
func (c *Console) Loader(message string) *Loader {
	return &Loader{console: c, message: normalizeTransientMessage(message)}
}

// NewLoader constructs a loader using a snapshot of the current default console.
// It does not start the loader.
// @group Loaders
func NewLoader(message string) *Loader {
	return Default().Loader(message)
}

// Start begins the loader and is harmless when called more than once.
// Animated loaders can return ErrTransientActive when another live display owns the same console.
// @group Loaders
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
// @group Loaders
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
// @group Loaders
func (l *Loader) Stop() {
	l.finish(loaderFinishStop, "")
}

// Success completes the loader with a success message.
// An empty message reuses the loader's current message.
// @group Loaders
func (l *Loader) Success(message string) {
	l.finish(loaderFinishSuccess, message)
}

// Warn completes the loader with a warning message.
// An empty message reuses the loader's current message.
// @group Loaders
func (l *Loader) Warn(message string) {
	l.finish(loaderFinishWarn, message)
}

// Fail completes the loader with an error message on stderr.
// An empty message reuses the loader's current message.
// @group Loaders
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
	frame := singleLineLayoutText(frames[l.frame%len(frames)])
	messageWidth := max(l.console.Width()-VisibleWidth(frame)-1, 1)
	message := l.console.truncate(l.message, messageWidth)
	return clearTransientLine + l.console.Colorize(ColorGreen, frame) + " " + message
}
