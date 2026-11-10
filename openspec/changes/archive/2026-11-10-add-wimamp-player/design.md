## Context

The repository holds an unverified Go draft: `main.go` (CLI), `mpv.go` (IPC client, daemon start, status), `sock_unix.go`/`sock_windows.go` (endpoint and process detaching), `keys.go` with a test (bindings, sequences), and `tui.go` (Bubbletea). The keys test passes, and the binary builds for darwin, linux and windows. The CLI smoke test failed: every command reported "mpv started but its socket never appeared", even against an mpv already listening on the same path, and each failed command started another mpv, leaving nine running.

## Goals / Non-Goals

**Goals:**
- Wimamp contains no audio code. Formats, decoding, gapless playback and device handling all belong to the engine.
- Every interface is a thin client of one shared player state.
- A single binary (around 10 MB) and near-zero CPU when idle.

**Non-Goals:**
- Wimamp owning any playback state beyond what the engine holds. Restoring the last playlist on restart is deferred.
- Reading tags or durations for entries that aren't playing (deferred to the library work).
- Fully static Linux builds. The window needs the system's GL/X11 libraries.

## Decisions

**mpv as the engine and the daemon.** Wimamp starts `mpv --idle=yes --no-video --no-terminal --force-window=no --input-ipc-server=<endpoint>` detached (`setsid` on Unix; `DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP` on Windows). mpv's JSON IPC already provides the playlist, properties, shuffle and loop. The alternative, native decoding (Rust symphonia/rodio, or Go beep), gives a truly single-file install but means owning format quirks, gapless playback and device switching. A wimamp daemon in front of mpv was also rejected for now: it adds a process and a protocol with no current requirement that needs it. The user's `mpv.conf` still loads (so their audio settings apply), and wimamp's command-line flags override it.

**Endpoint location.** On Unix, `$XDG_RUNTIME_DIR/wimamp-<uid>.sock`, falling back to the user cache directory (`~/Library/Caches` on macOS). `$TMPDIR` is avoided because launchers and hotkey daemons may not inherit it. On Windows, the named pipe `\\.\pipe\wimamp`. `WIMAMP_SOCKET` overrides both. macOS limits socket paths to 104 bytes, so an overly long path must fail with a clear error rather than a timeout.

**Single-instance start.** `ensureDaemon` takes an exclusive lock file next to the endpoint (flock on Unix, LockFileEx on Windows) before deciding to spawn. Holding the lock, it retries a connect. If the connect fails and a socket file exists, it removes the stale socket, starts mpv once, and waits up to 5 s. A failed start kills the child it started. This fixes the pile-up seen in the smoke test. The smoke-test failure itself came from a socket path of 120 bytes (the scratch directory): macOS limits Unix socket paths to 104 bytes, and mpv fails to bind silently, so wimamp now rejects paths that are too long up front.

**IPC client.** One connection per call, writing a batch of commands from a goroutine while reading replies matched by `request_id`, and skipping event lines. Per-call connections keep the clients stateless; at a 500 ms poll rate the cost is negligible. Batching keeps `add` of thousands of files to a single connection.

**Stop position kept in mpv.** `stop keep-playlist` resets mpv's playlist position to -1, so `stop` saves the position in `user-data/wimamp/pos` first, and `resume` reads it back (clamped to the playlist length). Shuffle state lives in `user-data/wimamp/shuffle` for the same reason: `playlist-shuffle` is a one-shot reorder, and the ephemeral window needs to read the current state. Wimamp therefore still holds no state of its own.

**Status by polling.** The TUI polls a batch of properties every 500 ms and the window every 250 ms (fast enough for its scrolling title), instead of subscribing to `observe_property` events. It's simpler, and fast enough for a clock that shows whole seconds. A hidden window skips polling; its ticker still wakes four times a second to check that, which measures as 0.0% CPU.

**One ordered player queue.** Every player call and every state fetch from the TUI or the window runs on a single worker goroutine in submission order. Without it, a fast "stop then play" could run as "play then stop", two shuffle toggles could read the same initial state, and a slow poll could overwrite the state an action just produced (and an index-based `dd` would then remove the wrong entry). The window enqueues without blocking, because it holds its UI lock while enqueueing and the worker needs that lock to store results; a full queue (only if mpv hangs) shows "player busy". The TUI runs the commands of one burst of keys with `tea.Sequence`.

**Lock files per socket.** Start locks are named after the socket (`<socket>.daemon.lock`, `<socket>.gui.lock`), so instances on different `WIMAMP_SOCKET`s never block each other.

**Keybindings.** A flat `action = keys` file, parsed without a TOML or YAML dependency. Key names follow Bubbletea's key strings (`ctrl+d`, `enter`, `space`). The window maps Ebitengine key codes to the same names, so one keys.conf serves both.

**Window: Ebitengine.** The window is pixel-art drawn in code, which matches an immediate-mode 2D engine better than widget toolkits (Fyne, Gio) would. It needs no cgo on macOS or Windows. `ebiten.SetScreenClearedEveryFrame(false)` with a dirty flag skips repaints, and `FPSModeVsyncOffMinimum` runs frames only on input or on `ScheduleFrame` (called after every state poll, so 4 frames/s). Both of those APIs are deprecated since Ebitengine 2.5, but the replacement (vsync on/off plus a low TPS) measured 4–5% CPU idle against 0.6% for minimum mode, because every frame is still presented. If they are removed in a later version, revisit this. The window is undecorated, 360×408 points, with one logical pixel per point. On Retina, Ebitengine draws it at 2x with integer scaling, so the pixel font stays crisp. The first version used 2 points per pixel, which the user found too big: playlist titles were cut off. The text uses a small permissively licensed bitmap font embedded with `go:embed`. The look is original: bevelled grey panels, a green LCD, and a green-on-black list. Rejected: a webview (heavier, adds a browser runtime) and per-OS native UIs (three codebases).

**Window single instance and closing: a resident, hidden window.** A fresh window process needs about 450 ms before its window appears (GLFW plus Metal setup; wimamp's own work is about 2 ms), which the user felt as 1–2 s when launched through Raycast. So on macOS the first `wimamp gui` stays resident: closing (Esc, the close key, focus loss) calls `SetWindowVisible(false)` and stops the state polling. It listens on `<socket>.gui.sock`, where a later `wimamp gui` sends `show` and exits (window visible again in about 150 ms), and `wimamp quit` sends `quit`. Through purego's objc bindings (already an Ebitengine dependency) the process sets the accessory activation policy (no Dock icon or Cmd-Tab entry) and activates itself on show, which needs AppKit calls on the main thread, hence `RunGameOptions{SingleThread: true}`. A lock file serializes starts, and a stale control socket is removed under the lock. Windows has no named-pipe server in the standard library, so there the window keeps exiting on close. Alternatives considered: starting at full frame rate to show the window sooner (measured: no gain) and a per-launch prewarm (it still pays the 450 ms).

**File picker: ncruces/zenity.** It's pure Go: osascript on macOS, Win32 on Windows, zenity/kdialog on Linux. The picker supports multiple files or a directory. Results go through the same `collect` and `load(…, "append")` as the CLI.

## Risks / Trade-offs

- mpv must be installed separately. → A clear error, and install lines in the README for brew, apt and scoop.
- On macOS, the background mpv's privacy permissions are attributed to the app that started it (the "responsible process"). Started from a terminal that never granted the microphone prompt CoreAudio triggers, mpv plays silently. → The README documents the fix (allow the app under Privacy & Security → Microphone, or start the player from another app first). Starting mpv through launchd would avoid this, at the cost of more machinery.
- Closing on focus loss misfires with focus-follows-mouse window managers. → Add a `gui.close_on_blur` toggle in keys.conf's sibling config, if anyone needs it.
- The Windows named pipe and the Linux window are only compile-checked. → Manual checks are listed in tasks.
- Polling misses sub-second state changes. This is acceptable for this UI.
- zenity on Linux needs the `zenity` or `kdialog` binary for ADD. → The error is shown in the window, and the CLI `add` still works.
