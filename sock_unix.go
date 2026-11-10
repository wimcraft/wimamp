//go:build !windows

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// socketPath avoids $TMPDIR so terminals, Raycast and hotkey daemons all agree on it.
func socketPath() string {
	if p := os.Getenv("WIMAMP_SOCKET"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir, _ = os.UserCacheDir()
		os.MkdirAll(dir, 0o700)
	}
	return filepath.Join(dir, fmt.Sprintf("wimamp-%d.sock", os.Getuid()))
}

func dial() (io.ReadWriteCloser, error) {
	return net.DialTimeout("unix", socketPath(), time.Second)
}

// detach puts mpv in its own session so closing the terminal doesn't kill playback.
func detach(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

// endpointErr catches paths longer than sun_path (104 bytes on macOS), which mpv fails on silently.
// The window's socket is the longer of the two, so checking it covers both.
func endpointErr() error {
	if p := guiSocketPath(); len(p) >= 104 {
		return fmt.Errorf("socket path %s is %d bytes, the OS limit is 103 (set WIMAMP_SOCKET to a shorter path)", p, len(p))
	}
	return nil
}

func removeStale() { removeSocket(socketPath()) }

// removeSocket deletes p only if it is a socket, so a mistyped WIMAMP_SOCKET can't delete a real file.
func removeSocket(p string) {
	if fi, err := os.Lstat(p); err == nil && fi.Mode()&os.ModeSocket != 0 {
		os.Remove(p)
	}
}

// lock takes an exclusive per-user lock named name; wait=false fails at once if it is held.
func lock(name string, wait bool) (func(), error) {
	p := strings.TrimSuffix(socketPath(), ".sock") + "." + name + ".lock" // per socket: separate instances never collide
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	how := syscall.LOCK_EX
	if !wait {
		how |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		f.Close()
		return nil, err
	}
	return func() { f.Close() }, nil // closing the fd releases the flock
}

func guiSocketPath() string { return strings.TrimSuffix(socketPath(), ".sock") + ".gui.sock" }

func dialGUI() (net.Conn, error) { return net.DialTimeout("unix", guiSocketPath(), time.Second) }

// listenGUI is called holding the gui lock, so a leftover socket file is stale.
func listenGUI() (net.Listener, error) {
	removeSocket(guiSocketPath())
	return net.Listen("unix", guiSocketPath())
}
