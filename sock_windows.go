//go:build windows

package main

import (
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

func socketPath() string {
	if p := os.Getenv("WIMAMP_SOCKET"); p != "" {
		return p
	}
	return `\\.\pipe\wimamp`
}

func dial() (io.ReadWriteCloser, error) { return os.OpenFile(socketPath(), os.O_RDWR, 0) }

// mpvPath finds mpv on PATH, falling back to where the winget package (shinchiro.mpv) installs it:
// that installer doesn't add itself to PATH.
func mpvPath() string {
	if p, err := exec.LookPath("mpv"); err == nil {
		return p
	}
	for _, env := range []string{"ProgramFiles", "ProgramW6432"} { // ProgramW6432: 32-bit callers on 64-bit Windows
		if dir := os.Getenv(env); dir != "" {
			if p := filepath.Join(dir, "MPV Player", "mpv.exe"); fileExists(p) {
				return p
			}
		}
	}
	return "mpv" // not found: let exec report it
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP: mpv outlives the console that started it.
func detach(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x8 | 0x200} }

func endpointErr() error { return nil }

func removeStale() {} // named pipes vanish with their server

// lock takes an exclusive per-user lock named name; wait=false fails at once if it is held.
func lock(name string, wait bool) (func(), error) {
	dir, _ := os.UserCacheDir()
	f, err := os.OpenFile(filepath.Join(dir, "wimamp-"+name+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK)
	if !wait {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	if err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, new(windows.Overlapped)); err != nil {
		f.Close()
		return nil, err
	}
	return func() { f.Close() }, nil
}

// shortcut: no resident window on Windows (no named-pipe server in the stdlib); the window exits on close.
func dialGUI() (net.Conn, error)       { return nil, errors.New("unsupported") }
func listenGUI() (net.Listener, error) { return nil, errors.New("unsupported") }
