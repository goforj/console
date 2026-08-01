package console

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// loaderTestWriter records complete writes and exposes a descriptor for terminal-policy tests.
type loaderTestWriter struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	writes chan string
	fd     uintptr
}

// newLoaderTestWriter creates a writer with enough event capacity for one loader test.
func newLoaderTestWriter(fd uintptr) *loaderTestWriter {
	return &loaderTestWriter{
		writes: make(chan string, 64),
		fd:     fd,
	}
}

// Write records one output operation and publishes it after the bytes are visible to readers.
func (w *loaderTestWriter) Write(value []byte) (int, error) {
	copyValue := string(append([]byte(nil), value...))
	w.mu.Lock()
	_, _ = w.buffer.Write(value)
	w.mu.Unlock()
	w.writes <- copyValue
	return len(value), nil
}

// Fd returns the synthetic descriptor used by the injected terminal detector.
func (w *loaderTestWriter) Fd() uintptr {
	return w.fd
}

// String returns a race-safe snapshot of all output written so far.
func (w *loaderTestWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

// fakeLoaderTicker advances only when a test explicitly supplies a tick.
type fakeLoaderTicker struct {
	ticks   chan time.Time
	stopped chan struct{}
	once    sync.Once
}

// newFakeLoaderTicker creates a manually controlled loader ticker.
func newFakeLoaderTicker() *fakeLoaderTicker {
	return &fakeLoaderTicker{
		ticks:   make(chan time.Time, 16),
		stopped: make(chan struct{}),
	}
}

// Ticks returns the manually driven tick stream.
func (t *fakeLoaderTicker) Ticks() <-chan time.Time {
	return t.ticks
}

// Stop records that the loader animation released its ticker.
func (t *fakeLoaderTicker) Stop() {
	t.once.Do(func() {
		close(t.stopped)
	})
}

// Tick advances the fake ticker without sleeping.
func (t *fakeLoaderTicker) Tick() {
	t.ticks <- time.Time{}
}

// blockingLoaderTicker holds ticker shutdown open so tests can inspect join ordering.
type blockingLoaderTicker struct {
	ticks        chan time.Time
	stopStarted  chan struct{}
	releaseStop  chan struct{}
	stopFinished chan struct{}
}

// newBlockingLoaderTicker creates a ticker whose Stop waits for an explicit release.
func newBlockingLoaderTicker() *blockingLoaderTicker {
	return &blockingLoaderTicker{
		ticks:        make(chan time.Time),
		stopStarted:  make(chan struct{}),
		releaseStop:  make(chan struct{}),
		stopFinished: make(chan struct{}),
	}
}

// Ticks returns the controlled tick stream.
func (t *blockingLoaderTicker) Ticks() <-chan time.Time {
	return t.ticks
}

// Stop exposes shutdown ordering and waits until the test permits cleanup to finish.
func (t *blockingLoaderTicker) Stop() {
	close(t.stopStarted)
	<-t.releaseStop
	close(t.stopFinished)
}

// newLoaderTestConsole creates a deterministic ASCII console and its controlled ticker.
func newLoaderTestConsole(terminal bool) (*Console, *loaderTestWriter, *loaderTestWriter, *fakeLoaderTicker) {
	colorEnabled := false
	unicodeEnabled := false
	animationsEnabled := true
	marks := ASCIIMarks()
	marks.SpinnerFrames = []string{"1", "2"}
	stdout := newLoaderTestWriter(1)
	stderr := newLoaderTestWriter(2)
	console := New(Config{
		Stdout:            stdout,
		Stderr:            stderr,
		ColorEnabled:      &colorEnabled,
		UnicodeEnabled:    &unicodeEnabled,
		AnimationsEnabled: &animationsEnabled,
		Marks:             &marks,
		Width:             80,
		LoaderInterval:    time.Hour,
		IsTerminal:        func(int) bool { return terminal },
	})
	ticker := newFakeLoaderTicker()
	console.newTicker = func(time.Duration) loaderTicker { return ticker }
	return console, stdout, stderr, ticker
}

// waitForLoaderWrite waits for one expected asynchronous render without relying on a sleep.
func waitForLoaderWrite(t *testing.T, writer *loaderTestWriter) string {
	t.Helper()
	select {
	case value := <-writer.writes:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for loader output")
		return ""
	}
}

// waitForLoaderTickerStop waits until an animation has released its fake ticker.
func waitForLoaderTickerStop(t *testing.T, ticker *fakeLoaderTicker) {
	t.Helper()
	select {
	case <-ticker.stopped:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for loader ticker to stop")
	}
}

// waitForLoaderFrame waits until the animation goroutine has consumed a controlled tick.
func waitForLoaderFrame(t *testing.T, loader *Loader, frame int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		loader.mu.Lock()
		current := loader.frame
		loader.mu.Unlock()
		if current == frame {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for loader frame %d; current frame is %d", frame, current)
		}
		runtime.Gosched()
	}
}

// requireNoLoaderWrite verifies that a synchronous operation did not enqueue output.
func requireNoLoaderWrite(t *testing.T, writer *loaderTestWriter) {
	t.Helper()
	select {
	case value := <-writer.writes:
		t.Fatalf("unexpected loader write %q", value)
	default:
	}
}

// drainLoaderWrites discards already-asserted write notifications without changing captured output.
func drainLoaderWrites(writer *loaderTestWriter) {
	for {
		select {
		case <-writer.writes:
		default:
			return
		}
	}
}

// TestLoaderAnimatedLifecycle verifies controlled frames, updates, normalization, and terminal idempotence.
func TestLoaderAnimatedLifecycle(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	loader := console.Loader("  build\napp  ")

	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"1 build app"; got != want {
		t.Fatalf("initial loader write = %q, want %q", got, want)
	}
	if err := loader.Start(); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	requireNoLoaderWrite(t, stdout)

	ticker.Tick()
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"2 build app"; got != want {
		t.Fatalf("ticked loader write = %q, want %q", got, want)
	}

	loader.Update(" package\r\nfiles ")
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"2 package files"; got != want {
		t.Fatalf("updated loader write = %q, want %q", got, want)
	}

	loader.Success("")
	waitForLoaderTickerStop(t, ticker)
	loader.Fail("ignored")
	loader.Warn("ignored")
	loader.Stop()
	loader.Update("ignored")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() after completion error = %v", err)
	}

	want := clearTransientLine + "1 build app" +
		clearTransientLine + "2 build app" +
		clearTransientLine + "2 package files" +
		clearTransientLine + "+ package files\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}

	drainLoaderWrites(stdout)
	ticker.Tick()
	requireNoLoaderWrite(t, stdout)
	if got := stdout.String(); got != want {
		t.Fatalf("output changed after completion tick: %q", got)
	}
}

// TestLoaderTerminalOutcomesVerifyFirstCallWins verifies every durable outcome and repeated terminal calls.
func TestLoaderTerminalOutcomesVerifyFirstCallWins(t *testing.T) {
	tests := []struct {
		name       string
		finish     func(*Loader)
		wantStdout string
		wantStderr string
	}{
		{
			name:       "stop",
			finish:     func(loader *Loader) { loader.Stop() },
			wantStdout: clearTransientLine + "1 work" + clearTransientLine,
		},
		{
			name:       "success",
			finish:     func(loader *Loader) { loader.Success("done") },
			wantStdout: clearTransientLine + "1 work" + clearTransientLine + "+ done\n",
		},
		{
			name:       "warn",
			finish:     func(loader *Loader) { loader.Warn("careful") },
			wantStdout: clearTransientLine + "1 work" + clearTransientLine + "! careful\n",
		},
		{
			name:       "fail",
			finish:     func(loader *Loader) { loader.Fail("broken") },
			wantStdout: clearTransientLine + "1 work" + clearTransientLine,
			wantStderr: "x broken\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			console, stdout, stderr, ticker := newLoaderTestConsole(true)
			loader := console.Loader("work")
			if err := loader.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			_ = waitForLoaderWrite(t, stdout)

			test.finish(loader)
			waitForLoaderTickerStop(t, ticker)
			loader.Stop()
			loader.Success("later success")
			loader.Warn("later warning")
			loader.Fail("later failure")

			if got := stdout.String(); got != test.wantStdout {
				t.Fatalf("stdout = %q, want %q", got, test.wantStdout)
			}
			if got := stderr.String(); got != test.wantStderr {
				t.Fatalf("stderr = %q, want %q", got, test.wantStderr)
			}
		})
	}
}

// TestLoaderRejectsConcurrentAnimationsAndCanRetry verifies exclusive transient ownership is recoverable.
func TestLoaderRejectsConcurrentAnimationsAndCanRetry(t *testing.T) {
	console, stdout, stderr, firstTicker := newLoaderTestConsole(true)
	secondTicker := newFakeLoaderTicker()
	tickers := []loaderTicker{firstTicker, secondTicker}
	var tickersMu sync.Mutex
	console.newTicker = func(time.Duration) loaderTicker {
		tickersMu.Lock()
		defer tickersMu.Unlock()
		if len(tickers) == 0 {
			panic("unexpected loader ticker request")
		}
		ticker := tickers[0]
		tickers = tickers[1:]
		return ticker
	}

	first := console.Loader("first")
	second := console.Loader("second")
	if err := first.Start(); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	if err := second.Start(); !errors.Is(err, ErrTransientActive) {
		t.Fatalf("second Start() error = %v, want ErrTransientActive", err)
	}
	requireNoLoaderWrite(t, stdout)

	first.Stop()
	waitForLoaderTickerStop(t, firstTicker)
	if err := second.Start(); err != nil {
		t.Fatalf("retried second Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	second.Success("complete")
	waitForLoaderTickerStop(t, secondTicker)

	want := clearTransientLine + "1 first" + clearTransientLine +
		clearTransientLine + "1 second" + clearTransientLine + "+ complete\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderOrdinaryOutputClearsAndRedraws verifies durable messages preserve the active transient line.
func TestLoaderOrdinaryOutputClearsAndRedraws(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	console.Info("durable")
	loader.Stop()
	waitForLoaderTickerStop(t, ticker)

	want := clearTransientLine + "1 working" +
		clearTransientLine + "i durable\n" + clearTransientLine + "1 working" +
		clearTransientLine
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderStartDefersBehindExistingPartialOutput verifies reverse-order line ownership.
func TestLoaderStartDefersBehindExistingPartialOutput(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	console.Print("prompt: ")
	if got, want := waitForLoaderWrite(t, stdout), "prompt: "; got != want {
		t.Fatalf("partial output = %q, want %q", got, want)
	}

	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	requireNoLoaderWrite(t, stdout)
	loader.Update("updated")
	requireNoLoaderWrite(t, stdout)

	console.Println("answer")
	if got, want := waitForLoaderWrite(t, stdout), "answer\n"; got != want {
		t.Fatalf("completed partial output = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"1 updated"; got != want {
		t.Fatalf("deferred loader output = %q, want %q", got, want)
	}

	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	want := "prompt: answer\n" + clearTransientLine + "1 updated" + clearTransientLine
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderStopPreservesPreexistingPartialOutput verifies a hidden loader does not complete another owner's line.
func TestLoaderStopPreservesPreexistingPartialOutput(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	console.Print("prefix")
	if got, want := waitForLoaderWrite(t, stdout), "prefix"; got != want {
		t.Fatalf("partial output = %q, want %q", got, want)
	}

	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	requireNoLoaderWrite(t, stdout)
	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	requireNoLoaderWrite(t, stdout)

	console.Println(" suffix")
	if got, want := waitForLoaderWrite(t, stdout), " suffix\n"; got != want {
		t.Fatalf("completed partial output = %q, want %q", got, want)
	}
	if got, want := stdout.String(), "prefix suffix\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderCompletionSeparatesPreexistingPartialOutput keeps durable status on its own physical line.
func TestLoaderCompletionSeparatesPreexistingPartialOutput(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	console.Print("prefix")
	if got, want := waitForLoaderWrite(t, stdout), "prefix"; got != want {
		t.Fatalf("partial output = %q, want %q", got, want)
	}

	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	requireNoLoaderWrite(t, stdout)
	loader.Success("done")
	waitForLoaderTickerStop(t, ticker)
	if got, want := waitForLoaderWrite(t, stdout), "\n"; got != want {
		t.Fatalf("partial-line completion = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), "+ done\n"; got != want {
		t.Fatalf("durable completion = %q, want %q", got, want)
	}

	if got, want := stdout.String(), "prefix\n+ done\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderPartialOutputPausesTransientRendering verifies prompts and incremental output stay visible.
func TestLoaderPartialOutputPausesTransientRendering(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	console.Print("prompt: ")
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine; got != want {
		t.Fatalf("partial output clear = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), "prompt: "; got != want {
		t.Fatalf("partial output = %q, want %q", got, want)
	}

	loader.Update("updated")
	requireNoLoaderWrite(t, stdout)

	console.Println("answer")
	if got, want := waitForLoaderWrite(t, stdout), "answer\n"; got != want {
		t.Fatalf("completed partial output = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"1 updated"; got != want {
		t.Fatalf("resumed loader output = %q, want %q", got, want)
	}

	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	want := clearTransientLine + "1 working" + clearTransientLine + "prompt: answer\n" +
		clearTransientLine + "1 updated" + clearTransientLine
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderStderrDoesNotCompletePartialStdout verifies separate streams keep independent line state.
func TestLoaderStderrDoesNotCompletePartialStdout(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	console.Print("prompt: ")
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine; got != want {
		t.Fatalf("partial output clear = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), "prompt: "; got != want {
		t.Fatalf("partial output = %q, want %q", got, want)
	}

	console.Error("failed")
	requireNoLoaderWrite(t, stdout)
	if got, want := stderr.String(), "x failed\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}

	console.Println("answer")
	if got, want := waitForLoaderWrite(t, stdout), "answer\n"; got != want {
		t.Fatalf("completed partial output = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"1 working"; got != want {
		t.Fatalf("resumed loader output = %q, want %q", got, want)
	}

	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	want := clearTransientLine + "1 working" + clearTransientLine + "prompt: answer\n" +
		clearTransientLine + "1 working" + clearTransientLine
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestLoaderDoesNotEraseBlockingPrompt verifies ticker frames are suppressed while input is pending.
func TestLoaderDoesNotEraseBlockingPrompt(t *testing.T) {
	console, stdout, stderr, ticker := newLoaderTestConsole(true)
	interactive := true
	console.interactiveEnabled = &interactive
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	console.stdin.Reset(reader)
	console.stdinSource = reader

	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	type result struct {
		answer string
		err    error
	}
	resultChannel := make(chan result, 1)
	go func() {
		answer, err := console.Ask("Name")
		resultChannel <- result{answer: answer, err: err}
	}()

	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine; got != want {
		t.Fatalf("prompt clear = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), "> Name: "; got != want {
		t.Fatalf("prompt output = %q, want %q", got, want)
	}
	messageStarted := make(chan struct{})
	messageDone := make(chan struct{})
	go func() {
		close(messageStarted)
		console.Info("background")
		close(messageDone)
	}()
	<-messageStarted
	select {
	case <-messageDone:
		t.Fatal("background message completed while the prompt still owned output")
	default:
	}
	requireNoLoaderWrite(t, stdout)

	ticker.Tick()
	waitForLoaderFrame(t, loader, 1)
	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	requireNoLoaderWrite(t, stdout)
	if got, want := stdout.String(), clearTransientLine+"1 working"+clearTransientLine+"> Name: "; got != want {
		t.Fatalf("stdout while prompt blocked = %q, want %q", got, want)
	}

	if _, err := io.WriteString(writer, "Ada\n"); err != nil {
		t.Fatalf("write prompt answer: %v", err)
	}
	select {
	case got := <-resultChannel:
		if got.err != nil {
			t.Fatalf("Ask() error = %v", got.err)
		}
		if got.answer != "Ada" {
			t.Fatalf("Ask() = %q, want %q", got.answer, "Ada")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Ask() to return")
	}
	select {
	case <-messageDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the background message after prompt completion")
	}
	if got, want := waitForLoaderWrite(t, stdout), "i background\n"; got != want {
		t.Fatalf("deferred background output = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderStopWaitsForTickerShutdown verifies Stop joins the complete animation goroutine cleanup.
func TestLoaderStopWaitsForTickerShutdown(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(true)
	ticker := newBlockingLoaderTicker()
	console.newTicker = func(time.Duration) loaderTicker { return ticker }
	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	loader.mu.Lock()
	done := loader.done
	loader.mu.Unlock()
	stopReturned := make(chan struct{})
	go func() {
		loader.Stop()
		close(stopReturned)
	}()

	select {
	case <-ticker.stopStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ticker shutdown to start")
	}
	select {
	case <-done:
		t.Error("loader animation reported completion before ticker shutdown finished")
	default:
	}
	select {
	case <-stopReturned:
		t.Error("Loader.Stop returned before ticker shutdown finished")
	default:
	}

	close(ticker.releaseStop)
	select {
	case <-ticker.stopFinished:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ticker shutdown to finish")
	}
	select {
	case <-stopReturned:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Loader.Stop to return")
	}

	if got, want := stdout.String(), clearTransientLine+"1 working"+clearTransientLine; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestLoaderRedirectedOutputUsesStableSemanticLines verifies forced animation never leaks controls to logs.
func TestLoaderRedirectedOutputUsesStableSemanticLines(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(false)
	tickerRequested := false
	console.newTicker = func(time.Duration) loaderTicker {
		tickerRequested = true
		return newFakeLoaderTicker()
	}

	first := console.Loader("  building\napp ")
	if err := first.Start(); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	first.Update(" packaging\r\nfiles ")
	first.Success("")
	first.Fail("ignored")

	second := console.Loader("checking")
	if err := second.Start(); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	second.Fail("")

	if tickerRequested {
		t.Fatal("redirected loader requested an animation ticker")
	}
	wantStdout := "- building app\n+ packaging files\n- checking\n"
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("stdout = %q, want %q", got, wantStdout)
	}
	if got, want := stderr.String(), "x checking\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if strings.ContainsAny(stdout.String()+stderr.String(), "\r\x1b") {
		t.Fatalf("redirected output contains terminal controls: %q", stdout.String()+stderr.String())
	}
}

// TestLoaderRenderingStaysWithinConfiguredWidth verifies frames and labels share one strict terminal-cell budget.
func TestLoaderRenderingStaysWithinConfiguredWidth(t *testing.T) {
	t.Parallel()

	for _, unicodeEnabled := range []bool{false, true} {
		for width := 1; width <= 96; width++ {
			console, _, _, _ := newLoaderTestConsole(true)
			console.unicodeEnabled = unicodeEnabled
			console.width = width
			console.marks.SpinnerFrames = []string{"界界"}
			loader := console.Loader("download long-lived artifacts")
			loader.state = loaderRunning
			loader.dynamic = true

			rendered := strings.TrimPrefix(loader.renderTransient(), clearTransientLine)
			if got := VisibleWidth(rendered); got > width {
				t.Fatalf("Unicode=%t width=%d rendered width = %d: %q", unicodeEnabled, width, got, rendered)
			}
			if strings.ContainsAny(rendered, "\r\n") {
				t.Fatalf("Unicode=%t width=%d rendered multiple lines: %q", unicodeEnabled, width, rendered)
			}
		}
	}
}

// TestNewLoaderSnapshotsDefaultConsole verifies later default changes cannot redirect an existing loader.
func TestNewLoaderSnapshotsDefaultConsole(t *testing.T) {
	previous := Default()
	t.Cleanup(func() {
		SetDefault(previous)
	})

	first, firstStdout, firstStderr, _ := newLoaderTestConsole(false)
	second, secondStdout, secondStderr, _ := newLoaderTestConsole(false)
	SetDefault(first)
	loader := NewLoader("work")
	SetDefault(second)

	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	loader.Success("done")

	if got, want := firstStdout.String(), "- work\n+ done\n"; got != want {
		t.Fatalf("first stdout = %q, want %q", got, want)
	}
	if got := firstStderr.String(); got != "" {
		t.Fatalf("first stderr = %q, want empty", got)
	}
	if got := secondStdout.String(); got != "" {
		t.Fatalf("second stdout = %q, want empty", got)
	}
	if got := secondStderr.String(); got != "" {
		t.Fatalf("second stderr = %q, want empty", got)
	}
}

// TestRealLoaderTickerAdapter verifies the production ticker exposes a usable channel and clean stop operation.
func TestRealLoaderTickerAdapter(t *testing.T) {
	ticker := newRealLoaderTicker(time.Hour)
	if ticker.Ticks() == nil {
		t.Fatal("Ticks() = nil, want a time channel")
	}
	ticker.Stop()
}

var _ io.Writer = (*loaderTestWriter)(nil)
