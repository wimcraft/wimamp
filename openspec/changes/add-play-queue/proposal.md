## Why

Winamp's Jump To File Extra (JTFE) made the queue a habit: find a song, queue it, keep listening, and the queued songs play next in the order picked. wimamp already has JTFE's search, but no way to say "this one next" without breaking the current playlist order.

## What Changes

- A play queue: `e` queues the selected track (or unqueues it), `ctrl+e` queues the highlighted search match and moves to the next one, keeping the search. Search results that matched a folder show it before the name. Queued tracks play next, in the order queued, then the playlist continues after the last of them, as in Winamp. The playlist order never changes.
- `next` (key, button and `wimamp next`) plays the first queued track when there is one.
- Queued rows show their place, `[1]`, `[2]`…, before the name, in the window and the TUI.
- A queue view instead of JTFE's queue manager window: `E` turns the playlist panel into the queue (title `WIMAMP QUEUE` in the window). There `enter` plays, `e` or `dd` unqueue without touching the playlist, `J` / `K` move a track later or earlier, `E` or Esc go back.
- The queue works with no wimamp interface open: a small Lua script embedded in the binary runs inside mpv and picks the queued track at each track change.

Out of scope: saving or loading the queue, saving search results as a playlist, a queue CLI command, JTFE's text formatting, skinning and SDK.

## Capabilities

### New Capabilities

### Modified Capabilities
- `playlists`: a play queue and a queue view.
- `track-search`: queueing from search, and showing the folder a result matched.
- `keybindings`: default keys for queueing, the queue view and moving within it.

## Impact

- New `queue.lua`, embedded and passed to mpv with `--script` when wimamp starts the player; a player started by an older wimamp loads it on the first queue change.
- `mpv.go`: queue state in mpv (`user-data/wimamp/queue`), entry ids in the playlist, `next` honoring the queue.
- `gui.go`, `tui.go`, `search.go`, `actions.go`, `keys.go`, `help.go`: keys, marks and the queue view.
- An integration test drives a private mpv when one is installed.
