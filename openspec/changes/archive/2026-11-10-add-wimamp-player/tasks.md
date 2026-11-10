## 1. Fix the player daemon

- [x] 1.1 Find out why connecting fails against a running mpv in the smoke test (path length, sandbox, client bug), and fix it
- [x] 1.2 Add the start lock and stale-socket cleanup to `ensureDaemon`. A failed start kills its child
- [x] 1.3 Fail fast with a clear error when the socket path is too long for the OS
- [x] 1.4 CLI smoke test against a silent mpv (`--ao=null`, isolated `WIMAMP_SOCKET`): every command in `cli-control` and `playlists`, and no stray mpv left afterwards
- [x] 1.5 Concurrency check: start 5 `wimamp status` calls in parallel with no player running, and confirm exactly one mpv

## 2. Verify CLI and keybindings against the specs

- [x] 2.1 `play` with paths, `resume` after `stop`, and `stop` keeps the playlist
- [x] 2.2 `add` folder ordering, extension filtering, the "no audio files found" error and the missing-path error
- [x] 2.3 `save`/`load`/`playlists` round trip. `load` of a missing playlist leaves the current one unchanged
- [x] 2.4 Update the default keys to match the keybindings spec (`o`, `x`, `c`, `v`, `b`, `z`, `pgdown`/`pgup`) and extend `keys_test.go`
- [x] 2.5 A keys.conf error message names the line number. An empty `action =` unbinds the action

## 3. Verify the TUI

- [x] 3.1 Run the TUI in a terminal and check the layout, the empty-playlist hint, and the initial selection on the playing entry
- [x] 3.2 An external `wimamp next` shows up within 1 s. `q` and `ctrl+c` leave playback running. `Q` quits the player
- [x] 3.3 A command error shows at the bottom and clears on the next key

## 4. Popup window

- [x] 4.1 Add Ebitengine and a bundled bitmap font. Add a `wimamp gui` subcommand with an undecorated window and low-FPS mode
- [x] 4.2 Draw the main panel: LCD time, scrolling title, transport buttons, seek bar, volume, shuffle/repeat toggles
- [x] 4.3 Draw the playlist panel: numbered rows, playing marker, selection highlight, scrolling, ADD/REM buttons
- [x] 4.4 Keyboard: map Ebitengine keys to keys.conf names and reuse the binding resolver
- [x] 4.5 Mouse: buttons, seek and volume drag, double-click to play, wheel scroll, drag the window by its title bar
- [x] 4.6 Shuffle and repeat backed by the engine's playlist shuffle and playlist loop
- [x] 4.7 ADD through the native picker (files or a folder), going through the same add rules as the CLI. REM removes the selected entry
- [x] 4.8 Single instance via a lock. Close on Esc, `q`, and focus loss. Playback continues after closing
- [x] 4.9 Manual check on macOS: a screenshot, every scenario in the popup-window spec, and idle CPU while paused
- [x] 4.10 ~~Compile check for windows. A Linux build on a Linux machine~~ deferred: the user is focusing on macOS for now (Windows still compiles)

## 5. Docs

- [x] 5.1 README: installing mpv (brew, apt, scoop), the commands, keys.conf actions, and hotkey examples for Raycast, Hammerspoon, i3 and AutoHotkey
- [x] 5.2 `devdocs/fyi.md`: the decision log
- [x] 5.3 ~~bd issues for deferred work~~ skipped: the user chose not to use bd for this project

## 6. Feedback from the first manual test

- [x] 6.1 Window: `ctrl+m`/`ctrl+i`/`ctrl+h`/`ctrl+[` act as enter/tab/backspace/Esc
- [x] 6.2 Key help: `?` overlay in the TUI and the window, and `wimamp keys`, generated from the active bindings
- [x] 6.3 Shared search matching and prompt logic, with unit tests
- [x] 6.4 Search prompt in the TUI
- [x] 6.5 Search prompt in the window (Esc closes the prompt, not the window)
- [x] 6.6 Find out why the TUI played no sound when started from one terminal app: macOS privacy (TCC). mpv's CoreAudio use was attributed to the terminal that started it, whose microphone prompt went unanswered. Fix: allow that app under Privacy & Security → Microphone, or start the player from another app first

## 7. Second round of feedback

- [x] 7.1 `wimamp tui` as an explicit alias for the TUI
- [x] 7.2 Smaller window: 1 point per logical pixel (crisp 2x on Retina), 360×408, 16 playlist rows of about 58 characters
- [x] 7.3 Raycast: a script command that opens the window detached (documented in the README)
- [x] 7.4 Fast reopen: on macOS the window stays resident and hidden after closing. `wimamp gui` shows it through a control socket (~150 ms instead of ~450 ms), with no Dock icon, and `wimamp quit` ends it

## 8. Fixes from the omp review (openai-codex/gpt-6.1-sol)

- [x] 8.1 `wimamp quit` ends a hidden resident window (the quit check ran after the hidden early-return)
- [x] 8.2 One ordered queue for all player calls and fetches from the TUI and the window (out-of-order actions, stale polls overwriting fresh state); volume drag and `Q` no longer block the UI thread
- [x] 8.3 A transport failure mid-call returns no data, so a dying player shows as an error instead of a zeroed "playing" state
- [x] 8.4 Play selected and `wimamp load` unpause; a second `stop` keeps the position saved by the first
- [x] 8.5 Control socket: a goroutine per client with a 1 s read deadline; stale sockets are removed only if they are sockets; the length check covers the window's socket; locks are named per socket
- [x] 8.6 Window: one file picker at a time; the mouse wheel moves the search selection while searching; a pending key prefix is cleared on hide; the selection sync on reopen redraws
- [x] 8.7 Unknown CLI commands are rejected before mpv starts; folders no longer pick up `.m3u` files
- [x] 8.8 Specs: polling cadence and idle-CPU limits made measurable; behavior after the player dies specified; play-selected-while-paused scenario
