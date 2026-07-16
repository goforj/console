//go:build windows

package console

import (
	"errors"
	"io"
	"strings"
	"sync"
	"syscall"
)

// enableVirtualTerminalProcessing identifies the console mode bit required for ANSI sequences.
const enableVirtualTerminalProcessing = 0x0004

// windowsConsoleModeMu prevents scoped mode restoration from racing across Console instances.
var windowsConsoleModeMu sync.Mutex

// setConsoleModeProcedure provides the Windows call that the syscall package does not wrap directly.
var setConsoleModeProcedure = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")

// terminalSupportsANSI reports whether a Windows console handle can use virtual-terminal processing.
// Restoring the original mode after the probe keeps capability detection invisible to embedding applications.
func terminalSupportsANSI(descriptor int) bool {
	windowsConsoleModeMu.Lock()
	defer windowsConsoleModeMu.Unlock()

	var mode uint32
	handle := syscall.Handle(descriptor)
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	if err := setConsoleMode(handle, mode|enableVirtualTerminalProcessing); err != nil {
		return false
	}
	return setConsoleMode(handle, mode) == nil
}

// writeTerminalString temporarily enables virtual-terminal processing for ANSI-bearing console writes.
// The original mode is restored before returning so library output does not permanently mutate shared console state.
func writeTerminalString(writer io.Writer, value string) (written int, err error) {
	if !strings.Contains(value, "\x1b") {
		return io.WriteString(writer, value)
	}
	descriptor, ok := writerDescriptor(writer)
	if !ok {
		return io.WriteString(writer, value)
	}

	windowsConsoleModeMu.Lock()
	defer windowsConsoleModeMu.Unlock()

	handle := syscall.Handle(descriptor)
	var mode uint32
	if getErr := syscall.GetConsoleMode(handle, &mode); getErr != nil || mode&enableVirtualTerminalProcessing != 0 {
		return io.WriteString(writer, value)
	}
	if enableErr := setConsoleMode(handle, mode|enableVirtualTerminalProcessing); enableErr != nil {
		return io.WriteString(writer, value)
	}
	defer func() {
		err = errors.Join(err, setConsoleMode(handle, mode))
	}()
	return io.WriteString(writer, value)
}

// setConsoleMode changes one Windows console handle mode and translates a zero return into an error.
func setConsoleMode(handle syscall.Handle, mode uint32) error {
	result, _, callErr := setConsoleModeProcedure.Call(uintptr(handle), uintptr(mode))
	if result != 0 {
		return nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return callErr
	}
	return syscall.EINVAL
}
