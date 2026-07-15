package console

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// atomicWriteRecorder records write call boundaries and detects concurrent entry.
type atomicWriteRecorder struct {
	active  atomic.Int32
	overlap atomic.Bool
	mu      sync.Mutex
	chunks  []string
}

// trackedStringer records when formatted output evaluates its value.
type trackedStringer struct {
	calls *int
}

// failingOutputWriter returns a stable failure so adapter behavior can be tested without operating-system I/O.
type failingOutputWriter struct {
	err error
}

// shortOutputWriter accepts only a prefix without reporting the required error.
type shortOutputWriter struct{}

// Write reports the configured failure without accepting bytes.
func (w failingOutputWriter) Write([]byte) (int, error) {
	return 0, w.err
}

// Write simulates a broken destination so the adapter can restore the io.Writer contract.
func (shortOutputWriter) Write(value []byte) (int, error) {
	return min(2, len(value)), nil
}

// String records one formatting evaluation before returning a stable value.
func (s trackedStringer) String() string {
	(*s.calls)++
	return "value"
}

// Write records one output chunk while widening the scheduling window for overlap detection.
func (w *atomicWriteRecorder) Write(value []byte) (int, error) {
	if !w.active.CompareAndSwap(0, 1) {
		w.overlap.Store(true)
	}
	for range 8 {
		runtime.Gosched()
	}
	w.mu.Lock()
	w.chunks = append(w.chunks, string(value))
	w.mu.Unlock()
	w.active.Store(0)
	return len(value), nil
}

// snapshot returns an isolated copy of the recorded write chunks.
func (w *atomicWriteRecorder) snapshot() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.chunks...)
}

// TestConsolePlainOutputPreservesPrintSemantics verifies exact ordinary-output formatting.
func TestConsolePlainOutputPreservesPrintSemantics(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	console := New(Config{Stdout: &stdout, Getenv: getenvFrom(nil)})

	console.Print("raw", 1)
	console.Printf(":%02d", 2)
	console.Println(" line", 3)
	console.NewLine()

	want := "raw1:02 line 3\n\n"
	if got := stdout.String(); got != want {
		t.Fatalf("ordinary output = %q, want %q", got, want)
	}
}

// TestConsoleOutputWritersUseConfiguredDestinations verifies both adapters preserve Print-style semantics.
func TestConsoleOutputWritersUseConfiguredDestinations(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	console := New(Config{Stdout: &stdout, Stderr: &stderr, Getenv: getenvFrom(nil)})

	stdoutCount, stdoutErr := io.WriteString(console.StdoutWriter(), "ordinary")
	stderrCount, stderrErr := io.WriteString(console.StderrWriter(), "failure")
	if stdoutErr != nil || stdoutCount != len("ordinary") {
		t.Fatalf("stdout Write() = (%d, %v), want (%d, nil)", stdoutCount, stdoutErr, len("ordinary"))
	}
	if stderrErr != nil || stderrCount != len("failure") {
		t.Fatalf("stderr Write() = (%d, %v), want (%d, nil)", stderrCount, stderrErr, len("failure"))
	}
	if got, want := stdout.String(), "ordinary"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "failure"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

// TestConsoleOutputWritersPreserveDestinationErrors verifies adapters honor the io.Writer contract.
func TestConsoleOutputWritersPreserveDestinationErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("write failed")
	console := New(Config{
		Stdout: failingOutputWriter{err: wantErr},
		Stderr: failingOutputWriter{err: wantErr},
		Getenv: getenvFrom(nil),
	})
	for name, writer := range map[string]io.Writer{
		"stdout": console.StdoutWriter(),
		"stderr": console.StderrWriter(),
	} {
		name, writer := name, writer
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			count, err := writer.Write([]byte("value"))
			if count != 0 || !errors.Is(err, wantErr) {
				t.Fatalf("Write() = (%d, %v), want (0, %v)", count, err, wantErr)
			}
			count, err = writer.Write(nil)
			if count != 0 || err != nil {
				t.Fatalf("Write(nil) = (%d, %v), want (0, nil)", count, err)
			}
		})
	}
}

// TestConsoleOutputWritersReportSilentShortWrites verifies invalid destination results become io.ErrShortWrite.
func TestConsoleOutputWritersReportSilentShortWrites(t *testing.T) {
	t.Parallel()

	console := New(Config{Stdout: shortOutputWriter{}, Getenv: getenvFrom(nil)})
	count, err := console.StdoutWriter().Write([]byte("value"))
	if count != 2 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Write() = (%d, %v), want (2, io.ErrShortWrite)", count, err)
	}
}

// TestPackageOutputWritersSnapshotDefault verifies existing adapters do not change destinations after SetDefault.
func TestPackageOutputWritersSnapshotDefault(t *testing.T) {
	previous := Default()
	t.Cleanup(func() {
		SetDefault(previous)
	})

	var firstStdout bytes.Buffer
	var firstStderr bytes.Buffer
	SetDefault(New(Config{Stdout: &firstStdout, Stderr: &firstStderr, Getenv: getenvFrom(nil)}))
	stdoutWriter := StdoutWriter()
	stderrWriter := StderrWriter()

	var secondStdout bytes.Buffer
	var secondStderr bytes.Buffer
	SetDefault(New(Config{Stdout: &secondStdout, Stderr: &secondStderr, Getenv: getenvFrom(nil)}))
	_, _ = io.WriteString(stdoutWriter, "first out")
	_, _ = io.WriteString(stderrWriter, "first err")
	_, _ = io.WriteString(StdoutWriter(), "second out")
	_, _ = io.WriteString(StderrWriter(), "second err")

	if got, want := firstStdout.String(), "first out"; got != want {
		t.Fatalf("captured stdout = %q, want %q", got, want)
	}
	if got, want := firstStderr.String(), "first err"; got != want {
		t.Fatalf("captured stderr = %q, want %q", got, want)
	}
	if got, want := secondStdout.String(), "second out"; got != want {
		t.Fatalf("current stdout = %q, want %q", got, want)
	}
	if got, want := secondStderr.String(), "second err"; got != want {
		t.Fatalf("current stderr = %q, want %q", got, want)
	}
}

// TestConsoleOutputWriterWaitsForPromptOwnership verifies adapters honor the prompt session lock.
func TestConsoleOutputWriterWaitsForPromptOwnership(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	console := New(Config{Stdout: &stdout, Getenv: getenvFrom(nil)})
	console.sessionMu.Lock()
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		_, _ = io.WriteString(console.StdoutWriter(), "after prompt")
		close(done)
	}()
	<-started
	for range 32 {
		runtime.Gosched()
	}
	select {
	case <-done:
		console.sessionMu.Unlock()
		t.Fatal("writer completed while a prompt owned the output session")
	default:
	}
	console.sessionMu.Unlock()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for writer after prompt ownership ended")
	}
	if got, want := stdout.String(), "after prompt"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// TestConsoleOutputWriterPreservesLoaderRendering verifies durable adapter writes clear and restore a transient line.
func TestConsoleOutputWriterPreservesLoaderRendering(t *testing.T) {
	console, stdout, _, ticker := newLoaderTestConsole(true)
	loader := console.Loader("working")
	if err := loader.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_ = waitForLoaderWrite(t, stdout)

	count, err := io.WriteString(console.StdoutWriter(), "durable\n")
	if count != len("durable\n") || err != nil {
		t.Fatalf("Write() = (%d, %v), want (%d, nil)", count, err, len("durable\n"))
	}
	loader.Stop()
	waitForLoaderTickerStop(t, ticker)

	want := clearTransientLine + "1 working" +
		clearTransientLine + "durable\n" + clearTransientLine + "1 working" +
		clearTransientLine
	if got := stdout.String(); got != want {
		t.Fatalf("loader-coordinated stdout = %q, want %q", got, want)
	}
}

// TestSharedOutputTracksOnePhysicalLine verifies aliased stdout and stderr share transient state.
func TestSharedOutputTracksOnePhysicalLine(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	console := New(Config{
		Stdout:         &output,
		Stderr:         &output,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
	})
	console.Print("partial ")
	if !console.partialLine {
		t.Fatal("partialLine = false after partial stdout write, want true")
	}
	console.Error("failed")
	if console.partialLine {
		t.Fatal("partialLine = true after shared stderr newline, want false")
	}
	if got, want := output.String(), "partial x failed\n"; got != want {
		t.Fatalf("shared output = %q, want %q", got, want)
	}
}

// TestConsoleSemanticMessagesUseExpectedMarksAndDestinations verifies every message and formatting variant.
func TestConsoleSemanticMessagesUseExpectedMarksAndDestinations(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	console := New(Config{
		Stdout:         &stdout,
		Stderr:         &stderr,
		ColorEnabled:   boolPointer(false),
		DebugEnabled:   boolPointer(true),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
	})

	console.Action("fetch")
	console.Actionf("fetch %d", 2)
	console.Info("note")
	console.Infof("note %d", 3)
	console.Success("done")
	console.Successf("done %d", 4)
	console.Warn("careful")
	console.Warnf("careful %d", 5)
	console.Debug("trace")
	console.Debugf("trace %d", 6)
	console.Error("failed")
	console.Errorf("failed %d", 7)

	wantStdout := "- fetch\n- fetch 2\ni note\ni note 3\n+ done\n+ done 4\n! careful\n! careful 5\n? trace\n? trace 6\n"
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("semantic stdout = %q, want %q", got, wantStdout)
	}
	wantStderr := "x failed\nx failed 7\n"
	if got := stderr.String(); got != wantStderr {
		t.Fatalf("semantic stderr = %q, want %q", got, wantStderr)
	}
}

// TestConsoleSemanticMessagesHangingIndentContinuationLines aligns multiline text beneath its first line.
func TestConsoleSemanticMessagesHangingIndentContinuationLines(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	marks := ASCIIMarks()
	marks.Info = "界"
	console := New(Config{
		Stdout:         &stdout,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(true),
		Marks:          &marks,
		Getenv:         getenvFrom(nil),
	})
	console.Info("first\r\nsecond\rthird\n")

	if got, want := stdout.String(), "界 first\n   second\n   third\n   \n"; got != want {
		t.Fatalf("multiline semantic output = %q, want %q", got, want)
	}
}

// TestConsoleSemanticMessagesBalanceMultilineStyles prevents caller metadata from coloring indentation or later output.
func TestConsoleSemanticMessagesBalanceMultilineStyles(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	console := New(Config{
		Stdout:         &stdout,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
	})
	message := ColorRed + "first\r\nsecond" + ColorReset + "\x1b[20C\nthird"
	console.Info(message)

	want := "i " + ColorRed + "first" + ColorReset + "\n" +
		"  " + ColorRed + "second" + ColorReset + "\n" +
		"  third\n"
	if got := stdout.String(); got != want {
		t.Fatalf("styled multiline semantic output = %q, want %q", got, want)
	}
}

// TestConsoleSemanticMessagesKeepSingleLineBytes preserves the established pass-through behavior for one-line messages.
func TestConsoleSemanticMessagesKeepSingleLineBytes(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	console := New(Config{
		Stdout:         &stdout,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
	})
	message := "before\x1b[20Cafter\a"
	console.Info(message)

	if got, want := stdout.String(), "i "+message+"\n"; got != want {
		t.Fatalf("single-line semantic output = %q, want %q", got, want)
	}
}

// TestConsoleMarksHonorStdoutColorCapability verifies the GoForj-compatible mark color policy.
func TestConsoleMarksHonorStdoutColorCapability(t *testing.T) {
	t.Parallel()

	stdout := &descriptorBuffer{descriptor: 61}
	stderr := &descriptorBuffer{descriptor: 62}
	marks := Marks{Action: "A", Info: "I", Success: "S", Warn: "W", Error: "E", Debug: "D"}
	console := New(Config{
		Stdout: stdout,
		Stderr: stderr,
		Marks:  &marks,
		Getenv: getenvFrom(nil),
		IsTerminal: func(descriptor int) bool {
			return descriptor == 61
		},
	})

	markTests := []struct {
		name string
		got  string
		want string
	}{
		{name: "action", got: console.ActionMark(), want: ColorGray + "A" + ColorReset},
		{name: "info", got: console.InfoMark(), want: ColorGray + "I" + ColorReset},
		{name: "success", got: console.SuccessMark(), want: ColorGreen + "S" + ColorReset},
		{name: "warn", got: console.WarnMark(), want: ColorYellow + "W" + ColorReset},
		{name: "error", got: console.ErrorMark(), want: ColorRed + "E" + ColorReset},
		{name: "debug", got: console.DebugMark(), want: ColorGray + "D" + ColorReset},
	}
	for _, test := range markTests {
		if test.got != test.want {
			t.Fatalf("%s mark = %q, want %q", test.name, test.got, test.want)
		}
	}

	console.Success("ok")
	console.Error("bad")
	if got, want := stdout.String(), ColorGreen+"S"+ColorReset+" ok\n"; got != want {
		t.Fatalf("colored semantic stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), ColorRed+"E"+ColorReset+" bad\n"; got != want {
		t.Fatalf("redirected semantic stderr = %q, want %q", got, want)
	}
}

// TestConsoleStyleAppliesOrderedANSISequences verifies styling, colorization, and no-op cases.
func TestConsoleStyleAppliesOrderedANSISequences(t *testing.T) {
	t.Parallel()

	styled := New(Config{Stdout: &bytes.Buffer{}, ColorEnabled: boolPointer(true), Getenv: getenvFrom(nil)})
	plain := New(Config{Stdout: &bytes.Buffer{}, ColorEnabled: boolPointer(false), Getenv: getenvFrom(nil)})

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "multiple styles", got: styled.Style("value", StyleBold, ColorBlue), want: StyleBold + ColorBlue + "value" + ColorReset},
		{name: "colorize", got: styled.Colorize(ColorRed, "value"), want: ColorRed + "value" + ColorReset},
		{name: "empty value", got: styled.Style("", StyleBold), want: ""},
		{name: "no styles", got: styled.Style("value"), want: "value"},
		{name: "color disabled", got: plain.Style("value", StyleBold, ColorBlue), want: "value"},
		{name: "colorize disabled", got: plain.Colorize(ColorRed, "value"), want: "value"},
	}

	for _, test := range tests {
		if test.got != test.want {
			t.Fatalf("%s = %q, want %q", test.name, test.got, test.want)
		}
	}
}

// TestDebugMessagesApplyOverrideAndEnvironmentPrecedence verifies all supported debug switches.
func TestDebugMessagesApplyOverrideAndEnvironmentPrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		override *bool
		env      map[string]string
		want     string
	}{
		{name: "explicit enabled", override: boolPointer(true), want: "? debug\n"},
		{name: "explicit disabled beats environment", override: boolPointer(false), env: map[string]string{"FORJ_DEBUG": "1", "APP_DEBUG": "1", "DEBUG": "1"}, want: ""},
		{name: "FORJ_DEBUG", env: map[string]string{"FORJ_DEBUG": "1"}, want: "? debug\n"},
		{name: "APP_DEBUG", env: map[string]string{"APP_DEBUG": "true"}, want: "? debug\n"},
		{name: "DEBUG", env: map[string]string{"DEBUG": "yes"}, want: "? debug\n"},
		{name: "zero is disabled", env: map[string]string{"FORJ_DEBUG": "0", "APP_DEBUG": "0", "DEBUG": "0"}, want: ""},
		{name: "lower priority can enable", env: map[string]string{"FORJ_DEBUG": "0", "APP_DEBUG": "1"}, want: "? debug\n"},
		{name: "empty environment", env: map[string]string{}, want: ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			console := New(Config{
				Stdout:         &stdout,
				ColorEnabled:   boolPointer(false),
				DebugEnabled:   test.override,
				UnicodeEnabled: boolPointer(false),
				Getenv:         getenvFrom(test.env),
			})
			console.Debug("debug")
			if got := stdout.String(); got != test.want {
				t.Fatalf("debug output = %q, want %q", got, test.want)
			}
		})
	}
}

// TestDebugfSkipsFormattingWhenDisabled verifies suppressed diagnostics have no formatting side effects.
func TestDebugfSkipsFormattingWhenDisabled(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	calls := 0
	console := New(Config{
		Stdout:         &stdout,
		ColorEnabled:   boolPointer(false),
		DebugEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
	})
	console.Debugf("debug %s", trackedStringer{calls: &calls})

	if calls != 0 {
		t.Fatalf("disabled Debugf formatted its arguments %d times", calls)
	}
	if stdout.Len() != 0 {
		t.Fatalf("disabled Debugf output = %q, want empty", stdout.String())
	}
}

// TestFatalMessagesWriteBeforeInjectedExit verifies exact output, status, and operation ordering without terminating tests.
func TestFatalMessagesWriteBeforeInjectedExit(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	var exitCodes []int
	var outputAtExit []string
	console := New(Config{
		Stderr:         &stderr,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
		Exit: func(code int) {
			exitCodes = append(exitCodes, code)
			outputAtExit = append(outputAtExit, stderr.String())
		},
	})

	console.Fatal("first")
	console.Fatalf("second %d", 2)

	wantOutput := "x first\nx second 2\n"
	if got := stderr.String(); got != wantOutput {
		t.Fatalf("fatal output = %q, want %q", got, wantOutput)
	}
	if want := []int{1, 1}; !reflect.DeepEqual(exitCodes, want) {
		t.Fatalf("fatal exit codes = %v, want %v", exitCodes, want)
	}
	wantSnapshots := []string{"x first\n", "x first\nx second 2\n"}
	if !reflect.DeepEqual(outputAtExit, wantSnapshots) {
		t.Fatalf("output observed by Exit = %q, want %q", outputAtExit, wantSnapshots)
	}
}

// TestPackageHelpersRouteThroughDefault verifies the complete package-level surface uses the installed console.
func TestPackageHelpersRouteThroughDefault(t *testing.T) {
	previous := Default()
	t.Cleanup(func() {
		SetDefault(previous)
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var exitCodes []int
	marks := Marks{
		Action:        "A",
		Info:          "I",
		Success:       "S",
		Warn:          "W",
		Error:         "E",
		Debug:         "D",
		Bullet:        "B",
		Pointer:       "P",
		SpinnerFrames: []string{"F"},
	}
	configured := New(Config{
		Stdout:             &stdout,
		Stderr:             &stderr,
		ColorEnabled:       boolPointer(true),
		DebugEnabled:       boolPointer(true),
		InteractiveEnabled: boolPointer(true),
		UnicodeEnabled:     boolPointer(false),
		Width:              123,
		Marks:              &marks,
		Getenv:             getenvFrom(nil),
		Exit: func(code int) {
			exitCodes = append(exitCodes, code)
		},
	})
	SetDefault(configured)

	if got := Default(); got != configured {
		t.Fatalf("Default() = %p, want %p", got, configured)
	}
	if got := Width(); got != 123 {
		t.Fatalf("Width() = %d, want 123", got)
	}
	if !IsInteractive() {
		t.Fatal("IsInteractive() = false, want true")
	}
	if !SupportsColor() {
		t.Fatal("SupportsColor() = false, want true")
	}
	if SupportsUnicode() {
		t.Fatal("SupportsUnicode() = true, want false")
	}

	markTests := []struct {
		name string
		got  string
		want string
	}{
		{name: "ActionMark", got: ActionMark(), want: ColorGray + "A" + ColorReset},
		{name: "InfoMark", got: InfoMark(), want: ColorGray + "I" + ColorReset},
		{name: "SuccessMark", got: SuccessMark(), want: ColorGreen + "S" + ColorReset},
		{name: "WarnMark", got: WarnMark(), want: ColorYellow + "W" + ColorReset},
		{name: "ErrorMark", got: ErrorMark(), want: ColorRed + "E" + ColorReset},
		{name: "DebugMark", got: DebugMark(), want: ColorGray + "D" + ColorReset},
	}
	for _, test := range markTests {
		if test.got != test.want {
			t.Fatalf("%s() = %q, want %q", test.name, test.got, test.want)
		}
	}

	if got, want := Style("styled", StyleBold, ColorBlue), StyleBold+ColorBlue+"styled"+ColorReset; got != want {
		t.Fatalf("Style() = %q, want %q", got, want)
	}
	if got, want := Colorize(ColorCyan, "cyan"), ColorCyan+"cyan"+ColorReset; got != want {
		t.Fatalf("Colorize() = %q, want %q", got, want)
	}

	Print("raw")
	Printf(":%d", 2)
	Println(":line", 3)
	NewLine()
	Action("action")
	Actionf("action %d", 4)
	Info("info")
	Infof("info %d", 5)
	Success("success")
	Successf("success %d", 6)
	Warn("warn")
	Warnf("warn %d", 7)
	Debug("debug")
	Debugf("debug %d", 8)
	Error("error")
	Errorf("error %d", 9)
	Fatal("fatal")
	Fatalf("fatal %d", 10)

	grayA := ColorGray + "A" + ColorReset
	grayI := ColorGray + "I" + ColorReset
	greenS := ColorGreen + "S" + ColorReset
	yellowW := ColorYellow + "W" + ColorReset
	grayD := ColorGray + "D" + ColorReset
	wantStdout := "raw:2:line 3\n\n" +
		grayA + " action\n" + grayA + " action 4\n" +
		grayI + " info\n" + grayI + " info 5\n" +
		greenS + " success\n" + greenS + " success 6\n" +
		yellowW + " warn\n" + yellowW + " warn 7\n" +
		grayD + " debug\n" + grayD + " debug 8\n"
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("package helper stdout = %q, want %q", got, wantStdout)
	}
	redE := ColorRed + "E" + ColorReset
	wantStderr := redE + " error\n" + redE + " error 9\n" + redE + " fatal\n" + redE + " fatal 10\n"
	if got := stderr.String(); got != wantStderr {
		t.Fatalf("package helper stderr = %q, want %q", got, wantStderr)
	}
	if want := []int{1, 1}; !reflect.DeepEqual(exitCodes, want) {
		t.Fatalf("package helper exit codes = %v, want %v", exitCodes, want)
	}
}

// TestConcurrentSemanticWritesRemainAtomic verifies complete lines and cross-destination serialization under load.
func TestConcurrentSemanticWritesRemainAtomic(t *testing.T) {
	t.Parallel()

	recorder := &atomicWriteRecorder{}
	console := New(Config{
		Stdout:         recorder,
		Stderr:         recorder,
		ColorEnabled:   boolPointer(false),
		UnicodeEnabled: boolPointer(false),
		Getenv:         getenvFrom(nil),
	})

	const writes = 128
	want := make(map[string]int, writes)
	var wait sync.WaitGroup
	wait.Add(writes)
	for index := range writes {
		index := index
		if index%2 == 0 {
			want[fmt.Sprintf("i info-%03d\n", index)]++
		} else {
			want[fmt.Sprintf("x error-%03d\n", index)]++
		}
		go func() {
			defer wait.Done()
			if index%2 == 0 {
				console.Infof("info-%03d", index)
				return
			}
			console.Errorf("error-%03d", index)
		}()
	}
	wait.Wait()

	if recorder.overlap.Load() {
		t.Fatal("writer observed overlapping Write calls, want serialized output")
	}
	chunks := recorder.snapshot()
	if len(chunks) != writes {
		t.Fatalf("recorded write calls = %d, want %d", len(chunks), writes)
	}
	got := make(map[string]int, writes)
	for _, chunk := range chunks {
		got[chunk]++
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("concurrent output chunks = %#v, want %#v", got, want)
	}
}
