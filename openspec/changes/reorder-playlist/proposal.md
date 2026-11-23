## Why

There is no way to change the playlist order from wimamp: `J` / `K` only move entries inside the queue view, and in the playlist they do nothing, which reads as a bug.

## What Changes

- In the playlist (TUI and popup window), `J` moves the selected entry one place down and `K` one place up; the selection follows it.
- In the queue view, `J` / `K` keep moving the entry within the queue.

## Capabilities

### New Capabilities

### Modified Capabilities
- `playlists`: entries can be moved within the playlist.
- `keybindings`: `J` / `K` work in the playlist, not only in the queue view.

## Impact

`actions.go`, `gui.go`, `tui.go`, `help.go`, README. Uses mpv's `playlist-move`; no new dependency. Archive after `add-tui-fzf-picker` (both change the bindings table).
