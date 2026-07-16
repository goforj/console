package console

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

// blockingProgressWriter holds the first write open so lifecycle ordering can be observed deterministically.
type blockingProgressWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	buffer  bytes.Buffer
}

// Write blocks the first output operation until the test releases it, then records every complete write.
func (w *blockingProgressWriter) Write(value []byte) (int, error) {
	w.once.Do(func() {
		close(w.started)
		<-w.release
	})
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(value)
}

// String returns a race-safe snapshot of the recorded output.
func (w *blockingProgressWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

// TestProgressAnimatedLifecycle verifies updates, clamping, completion, and first-outcome semantics.
func TestProgressAnimatedLifecycle(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(true)
	progress := console.Progress(10, "build\napp")

	progress.Set(2)
	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"build app [====--------------------]  20%"; got != want {
		t.Fatalf("initial progress write = %q, want %q", got, want)
	}
	if err := progress.Start(); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	requireNoLoaderWrite(t, stdout)

	progress.Add(50)
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"build app [========================] 100%"; got != want {
		t.Fatalf("clamped progress write = %q, want %q", got, want)
	}
	progress.Add(math.MinInt)
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"build app [------------------------]   0%"; got != want {
		t.Fatalf("negative progress write = %q, want %q", got, want)
	}
	progress.Update("package files")
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"package files [------------------------]   0%"; got != want {
		t.Fatalf("updated progress write = %q, want %q", got, want)
	}

	progress.Complete("")
	progress.Fail("ignored")
	progress.Stop()
	progress.Set(3)
	progress.Add(1)
	progress.Update("ignored")
	if err := progress.Start(); err != nil {
		t.Fatalf("Start() after completion error = %v", err)
	}

	wantSuffix := clearTransientLine + "+ package files\n"
	if got := stdout.String(); !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("stdout suffix = %q, want suffix %q", got, wantSuffix)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestProgressRedirectedOutputIsStable verifies redirected logs receive only start and final semantic lines.
func TestProgressRedirectedOutputIsStable(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(false)
	progress := console.Progress(4, "download")

	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	progress.Set(1)
	progress.Add(2)
	progress.Update("install")
	progress.Complete("done")

	if got, want := stdout.String(), "- download\n+ done\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

// TestProgressStepUpdatesAmountAndMessageAtomically verifies one logical step produces one coherent redraw.
func TestProgressStepUpdatesAmountAndMessageAtomically(t *testing.T) {
	console, stdout, _, _ := newLoaderTestConsole(true)
	progress := console.Progress(4, "prepare")
	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	drainLoaderWrites(stdout)

	progress.Step(2, "compile packages")
	want := clearTransientLine + "compile packages [============------------]  50%"
	if got := waitForLoaderWrite(t, stdout); got != want {
		t.Fatalf("Step() redraw = %q, want %q", got, want)
	}
	requireNoLoaderWrite(t, stdout)

	progress.Complete("")
	drainLoaderWrites(stdout)
	progress.Step(1, "ignored")
	requireNoLoaderWrite(t, stdout)
	if got := stdout.String(); !strings.HasSuffix(got, clearTransientLine+"+ compile packages\n") {
		t.Fatalf("completed output = %q, want Step message as the outcome", got)
	}
}

// TestRedirectedProgressAndLoaderDoNotContend verifies transient ownership is reserved for live terminal displays.
func TestRedirectedProgressAndLoaderDoNotContend(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(false)
	console.newTicker = func(time.Duration) loaderTicker {
		t.Fatal("redirected displays requested an animation ticker")
		return nil
	}

	progress := console.Progress(4, "progress")
	loader := console.Loader("loader")
	if err := progress.Start(); err != nil {
		t.Fatalf("progress Start() error = %v", err)
	}
	if err := loader.Start(); err != nil {
		t.Fatalf("loader Start() error = %v", err)
	}
	progress.Set(2)
	progress.Update("progress updated")
	loader.Update("loader updated")
	loader.Success("")
	progress.Fail("")

	if got, want := stdout.String(), "- progress\n- loader\n+ loader updated\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "x progress updated\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if got := stdout.String() + stderr.String(); strings.ContainsAny(got, "\r\x1b") {
		t.Fatalf("redirected output contains terminal controls: %q", got)
	}
}

// TestProgressFailAndStop verifies error routing and silent cancellation.
func TestProgressFailAndStop(t *testing.T) {
	t.Run("fail", func(t *testing.T) {
		console, stdout, stderr, _ := newLoaderTestConsole(true)
		progress := console.Progress(2, "work")
		if err := progress.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		_ = waitForLoaderWrite(t, stdout)
		progress.Fail("")
		if got, want := stdout.String(), clearTransientLine+"work [------------------------]   0%"+clearTransientLine; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
		if got, want := stderr.String(), "x work\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})

	t.Run("stop", func(t *testing.T) {
		console, stdout, stderr, _ := newLoaderTestConsole(true)
		progress := console.Progress(2, "work")
		if err := progress.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		_ = waitForLoaderWrite(t, stdout)
		progress.Stop()
		if got, want := stdout.String(), clearTransientLine+"work [------------------------]   0%"+clearTransientLine; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
		if got := stderr.String(); got != "" {
			t.Fatalf("stderr = %q, want empty", got)
		}
	})
}

// TestProgressRejectsInvalidTotalAndTransientContention verifies retryable start failures.
func TestProgressRejectsInvalidTotalAndTransientContention(t *testing.T) {
	console, stdout, _, ticker := newLoaderTestConsole(true)
	invalid := console.Progress(0, "invalid")
	if err := invalid.Start(); !errors.Is(err, errInvalidProgressTotal) {
		t.Fatalf("invalid Start() error = %v, want %v", err, errInvalidProgressTotal)
	}

	loader := console.Loader("loader")
	if err := loader.Start(); err != nil {
		t.Fatalf("loader Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	progress := console.Progress(3, "progress")
	if err := progress.Start(); !errors.Is(err, ErrTransientActive) {
		t.Fatalf("progress Start() error = %v, want ErrTransientActive", err)
	}
	loader.Stop()
	waitForLoaderTickerStop(t, ticker)
	if err := progress.Start(); err != nil {
		t.Fatalf("retried progress Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	progress.Stop()
}

// TestProgressPreventsLoaderContention verifies progress and loaders share one transient line.
func TestProgressPreventsLoaderContention(t *testing.T) {
	console, stdout, _, _ := newLoaderTestConsole(true)
	progress := console.Progress(3, "progress")
	if err := progress.Start(); err != nil {
		t.Fatalf("progress Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)
	loader := console.Loader("loader")
	if err := loader.Start(); !errors.Is(err, ErrTransientActive) {
		t.Fatalf("loader Start() error = %v, want ErrTransientActive", err)
	}
	progress.Stop()
}

// TestProgressRenderingAdaptsToCapabilities verifies Unicode, color, and narrow layouts.
func TestProgressRenderingAdaptsToCapabilities(t *testing.T) {
	t.Run("unicode color", func(t *testing.T) {
		colorEnabled := true
		unicodeEnabled := true
		animationsEnabled := true
		stdout := newLoaderTestWriter(1)
		console := New(Config{
			Stdout:            stdout,
			Stderr:            &bytes.Buffer{},
			ColorEnabled:      &colorEnabled,
			UnicodeEnabled:    &unicodeEnabled,
			AnimationsEnabled: &animationsEnabled,
			Width:             32,
			IsTerminal:        func(int) bool { return true },
		})
		progress := console.Progress(4, "copy")
		progress.Set(1)
		if err := progress.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		got := waitForLoaderWrite(t, stdout)
		want := clearTransientLine + "copy [" + ColorGreen + "█████" + ColorReset + ColorGray + "░░░░░░░░░░░░░░░" + ColorReset + "]  25%"
		if got != want {
			t.Fatalf("render = %q, want %q", got, want)
		}
		progress.Stop()
	})

	t.Run("compact", func(t *testing.T) {
		console, stdout, _, _ := newLoaderTestConsole(true)
		console.width = 12
		progress := console.Progress(4, "downloading files")
		progress.Set(2)
		if err := progress.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if got, want := waitForLoaderWrite(t, stdout), clearTransientLine+"downloa. 50%"; got != want {
			t.Fatalf("compact render = %q, want %q", got, want)
		}
		if got := VisibleWidth(strings.TrimPrefix(stdout.String(), clearTransientLine)); got > console.Width() {
			t.Fatalf("compact width = %d, maximum %d", got, console.Width())
		}
		progress.Stop()
	})
}

// TestProgressRenderingKeepsCompactPercentReadable verifies fixed bar geometry does not consume narrow output.
func TestProgressRenderingKeepsCompactPercentReadable(t *testing.T) {
	console, _, _, _ := newLoaderTestConsole(true)
	progress := console.Progress(100, "go")
	for _, test := range []struct {
		width int
		want  string
	}{
		{width: 2, want: "0%"},
		{width: 3, want: "0%"},
		{width: 5, want: "go 0%"},
	} {
		console.width = test.width
		if got := progress.renderLine("go", 0, 100); got != test.want {
			t.Fatalf("renderLine(width %d) = %q, want %q", test.width, got, test.want)
		}
	}
}

// TestProgressRenderingUsesFixedPercentField verifies bar frames do not reflow at digit boundaries.
func TestProgressRenderingUsesFixedPercentField(t *testing.T) {
	console, _, _, _ := newLoaderTestConsole(true)
	progress := console.Progress(100, "work")
	frames := []string{
		progress.renderLine("work", 9, 100),
		progress.renderLine("work", 10, 100),
		progress.renderLine("work", 100, 100),
	}
	for index, frame := range frames {
		if got, want := VisibleWidth(frame), 36; got != want {
			t.Fatalf("frame %d width = %d, want %d: %q", index, got, want, frame)
		}
	}
	if !strings.HasSuffix(frames[0], "   9%") || !strings.HasSuffix(frames[1], "  10%") || !strings.HasSuffix(frames[2], " 100%") {
		t.Fatalf("fixed percent fields = %q", frames)
	}
}

// TestProgressRenderingStaysWithinConfiguredWidth verifies every adaptive presentation respects terminal cells.
func TestProgressRenderingStaysWithinConfiguredWidth(t *testing.T) {
	t.Parallel()

	for _, unicodeEnabled := range []bool{false, true} {
		for width := 1; width <= 96; width++ {
			console, _, _, _ := newLoaderTestConsole(true)
			console.unicodeEnabled = unicodeEnabled
			console.width = width
			progress := console.Progress(100, "download long-lived artifacts")
			for _, current := range []int{0, 1, 50, 99, 100} {
				rendered := progress.renderLine(progress.message, current, progress.total)
				if got := VisibleWidth(rendered); got > width {
					t.Fatalf("Unicode=%t width=%d current=%d rendered width = %d: %q", unicodeEnabled, width, current, got, rendered)
				}
				if strings.ContainsAny(rendered, "\r\n") {
					t.Fatalf("Unicode=%t width=%d current=%d rendered multiple lines: %q", unicodeEnabled, width, current, rendered)
				}
			}
		}
	}
}

// TestProgressCoordinatesDurableAndPartialOutput verifies transient redraw follows console line ownership.
func TestProgressCoordinatesDurableAndPartialOutput(t *testing.T) {
	console, stdout, _, _ := newLoaderTestConsole(true)
	progress := console.Progress(2, "work")
	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	frame := waitForLoaderWrite(t, stdout)
	drainLoaderWrites(stdout)

	console.Info("note")
	want := frame + clearTransientLine + "i note\n" + frame
	if got := stdout.String(); got != want {
		t.Fatalf("durable coordination = %q, want %q", got, want)
	}

	console.Print("partial")
	progress.Set(1)
	if got, wantPartial := stdout.String(), want+clearTransientLine+"partial"; got != wantPartial {
		t.Fatalf("partial coordination = %q, want %q", got, wantPartial)
	}
	progress.Complete("done")
	if got, wantCompleted := stdout.String(), want+clearTransientLine+"partial\n+ done\n"; got != wantCompleted {
		t.Fatalf("completed partial output = %q, want %q", got, wantCompleted)
	}
}

// TestProgressCoordinatesSecretPrompts verifies hidden input pauses and resumes the latest transient frame.
func TestProgressCoordinatesSecretPrompts(t *testing.T) {
	console, stdout, stderr, _ := newLoaderTestConsole(true)
	interactive := true
	console.interactiveEnabled = &interactive
	secretStarted := make(chan struct{})
	releaseSecret := make(chan struct{})
	console.readSecret = func() (string, error) {
		close(secretStarted)
		<-releaseSecret
		return "s3cr3t", nil
	}

	progress := console.Progress(2, "work")
	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	initialFrame := waitForLoaderWrite(t, stdout)

	result := make(chan struct {
		secret string
		err    error
	}, 1)
	go func() {
		secret, err := console.AskSecret("Password")
		result <- struct {
			secret string
			err    error
		}{secret: secret, err: err}
	}()
	select {
	case <-secretStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for secret input")
	}
	if got, want := waitForLoaderWrite(t, stdout), clearTransientLine; got != want {
		t.Fatalf("prompt clear = %q, want %q", got, want)
	}
	if got, want := waitForLoaderWrite(t, stdout), "> Password: "; got != want {
		t.Fatalf("prompt output = %q, want %q", got, want)
	}

	progress.Set(1)
	progress.Update("updated")
	requireNoLoaderWrite(t, stdout)
	close(releaseSecret)

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("AskSecret() error = %v", got.err)
		}
		if got.secret != "s3cr3t" {
			t.Fatalf("AskSecret() = %q, want %q", got.secret, "s3cr3t")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for AskSecret()")
	}
	if got, want := waitForLoaderWrite(t, stdout), "\n"; got != want {
		t.Fatalf("secret newline = %q, want %q", got, want)
	}
	latestFrame := clearTransientLine + "updated [============------------]  50%"
	if got := waitForLoaderWrite(t, stdout); got != latestFrame {
		t.Fatalf("resumed progress frame = %q, want %q", got, latestFrame)
	}

	progress.Complete("done")
	want := initialFrame + clearTransientLine + "> Password: \n" + latestFrame + clearTransientLine + "+ done\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
	if strings.Contains(stdout.String(), "s3cr3t") {
		t.Fatalf("secret appeared in output: %q", stdout.String())
	}
}

// TestProgressConcurrentLifecycleWritesOneOutcome verifies concurrent callers preserve one start and one terminal result.
func TestProgressConcurrentLifecycleWritesOneOutcome(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		console, stdout, stderr, _ := newLoaderTestConsole(false)
		progress := console.Progress(1000, "work")

		start := make(chan struct{})
		errorsChannel := make(chan error, 24)
		var starters sync.WaitGroup
		for range 24 {
			starters.Add(1)
			go func() {
				defer starters.Done()
				<-start
				errorsChannel <- progress.Start()
			}()
		}
		close(start)
		starters.Wait()
		close(errorsChannel)
		for err := range errorsChannel {
			if err != nil {
				t.Fatalf("iteration %d: concurrent Start() error = %v", iteration, err)
			}
		}

		finish := make(chan struct{})
		var callers sync.WaitGroup
		for index := 0; index < 64; index++ {
			callers.Add(1)
			go func(index int) {
				defer callers.Done()
				<-finish
				switch index % 4 {
				case 0:
					progress.Add(1)
				case 1:
					progress.Set(index)
				case 2:
					progress.Update("updated")
				default:
					if index%8 == 3 {
						progress.Complete("complete")
					} else {
						progress.Fail("failed")
					}
				}
			}(index)
		}
		close(finish)
		callers.Wait()

		if got := strings.Count(stdout.String(), "- work\n"); got != 1 {
			t.Fatalf("iteration %d: start line count = %d, output %q", iteration, got, stdout.String())
		}
		outcomes := strings.Count(stdout.String(), "+ complete\n") + strings.Count(stderr.String(), "x failed\n")
		if outcomes != 1 {
			t.Fatalf("iteration %d: terminal outcome count = %d, stdout %q, stderr %q", iteration, outcomes, stdout.String(), stderr.String())
		}
		if got := stdout.String() + stderr.String(); strings.ContainsAny(got, "\r\x1b") {
			t.Fatalf("iteration %d: redirected output contains controls: %q", iteration, got)
		}
	}
}

// TestProgressStartPublishesRedirectedActionBeforeTerminalCalls verifies outcomes cannot overtake a blocked start line.
func TestProgressStartPublishesRedirectedActionBeforeTerminalCalls(t *testing.T) {
	tests := []struct {
		name       string
		finish     func(*Progress)
		wantStdout string
		wantStderr string
	}{
		{name: "complete", finish: func(progress *Progress) { progress.Complete("done") }, wantStdout: "- work\n+ done\n"},
		{name: "fail", finish: func(progress *Progress) { progress.Fail("failed") }, wantStdout: "- work\n", wantStderr: "x failed\n"},
		{name: "stop", finish: func(progress *Progress) { progress.Stop() }, wantStdout: "- work\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout := &blockingProgressWriter{started: make(chan struct{}), release: make(chan struct{})}
			stderr := &loaderTestWriter{writes: make(chan string, 4)}
			colorEnabled := false
			unicodeEnabled := false
			animationsEnabled := true
			console := New(Config{
				Stdout:            stdout,
				Stderr:            stderr,
				ColorEnabled:      &colorEnabled,
				UnicodeEnabled:    &unicodeEnabled,
				AnimationsEnabled: &animationsEnabled,
				IsTerminal:        func(int) bool { return false },
			})
			progress := console.Progress(1, "work")

			startDone := make(chan error, 1)
			go func() { startDone <- progress.Start() }()
			select {
			case <-stdout.started:
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for redirected start write")
			}
			if progress.mu.TryLock() {
				progress.mu.Unlock()
				t.Fatal("progress lifecycle lock was released before its start line was published")
			}

			finishDone := make(chan struct{})
			go func() {
				test.finish(progress)
				close(finishDone)
			}()
			select {
			case <-finishDone:
				t.Fatal("terminal call completed before the blocked start line")
			default:
			}

			close(stdout.release)
			if err := <-startDone; err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			select {
			case <-finishDone:
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for terminal call")
			}
			if got := stdout.String(); got != test.wantStdout {
				t.Fatalf("stdout = %q, want %q", got, test.wantStdout)
			}
			if got := stderr.String(); got != test.wantStderr {
				t.Fatalf("stderr = %q, want %q", got, test.wantStderr)
			}
		})
	}
}

// TestNewProgressSnapshotsDefault verifies global construction does not retarget an existing display.
func TestNewProgressSnapshotsDefault(t *testing.T) {
	previous := Default()
	t.Cleanup(func() { SetDefault(previous) })

	firstOut := &bytes.Buffer{}
	secondOut := &bytes.Buffer{}
	colorEnabled := false
	animationsEnabled := false
	unicodeEnabled := false
	first := New(Config{
		Stdout:            firstOut,
		Stderr:            firstOut,
		ColorEnabled:      &colorEnabled,
		UnicodeEnabled:    &unicodeEnabled,
		AnimationsEnabled: &animationsEnabled,
	})
	second := New(Config{
		Stdout:            secondOut,
		Stderr:            secondOut,
		ColorEnabled:      &colorEnabled,
		UnicodeEnabled:    &unicodeEnabled,
		AnimationsEnabled: &animationsEnabled,
	})
	SetDefault(first)
	progress := NewProgress(1, "work")
	SetDefault(second)

	if err := progress.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	progress.Complete("done")
	if got, want := firstOut.String(), "- work\n+ done\n"; got != want {
		t.Fatalf("first output = %q, want %q", got, want)
	}
	if got := secondOut.String(); got != "" {
		t.Fatalf("second output = %q, want empty", got)
	}
}

// TestProgressPercentAndClampCoverBoundaries verifies helpers avoid invalid lifecycle values.
func TestProgressPercentAndClampCoverBoundaries(t *testing.T) {
	if got := progressPercent(1, 3); got != 33 {
		t.Fatalf("progressPercent(1, 3) = %d, want 33", got)
	}
	if got := progressPercent(29, 100); got != 29 {
		t.Fatalf("progressPercent(29, 100) = %d, want 29", got)
	}
	if got := progressPercent(math.MaxInt-1, math.MaxInt); got != 99 {
		t.Fatalf("large progress percent = %d, want 99", got)
	}
	for _, test := range []struct {
		current int
		total   int
		want    int
	}{{-1, 3, 0}, {1, 0, 0}, {4, 3, 3}, {2, 3, 2}} {
		if got := clampProgressValue(test.current, test.total); got != test.want {
			t.Fatalf("clampProgressValue(%d, %d) = %d, want %d", test.current, test.total, got, test.want)
		}
	}
}
