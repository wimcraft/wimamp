## Context

mpv, not wimamp, moves from one track to the next: wimamp hands it a playlist and sends commands. The CLI exits right away, the TUI and the window may be closed, and outside macOS nothing of wimamp stays running. Whatever decides "queue first" must run at every track change regardless. See proposal.md for the motivation and specs/ for the rules.

## Goals / Non-Goals

**Goals:** Winamp's queue semantics; playlist order untouched; works with no interface open; still one binary and no dependency beyond mpv.

**Non-Goals:** persisting the queue across player restarts; a separate queue window; queue editing from the CLI.

## Decisions

**A Lua script inside mpv.** mpv runs Lua scripts in-process for as long as it runs, so the check can't be missed and has no lifecycle of its own. The script is embedded with `go:embed`, written to the user cache dir on each player start (so it always matches the binary) and passed with `--script`. A player started by an older wimamp has no script; the first queue change sees the missing marker property and sends `load-script`.

**Ids, not indices.** The queue is a list of mpv playlist entry ids in `user-data/wimamp/queue`. Ids survive reordering (shuffle), removal of other entries and additions, so a queued entry never turns into a different track. Ids no longer in the playlist are skipped and dropped.

**Where the switch happens.** The script remembers whether the last `end-file` was a natural end (`eof`). In the `on_before_start_file` hook, which runs before mpv opens the entry it chose, it pops the first queued id still in the playlist and, if that isn't the chosen entry, plays it instead; the wrong track never starts. At the end of the playlist mpv goes idle without starting anything, so an `idle-active` observer does the same there. A `start-file` handler drops whatever starts from the queue, which also covers a queued track picked by hand.

**`next` in Go.** The script can't tell `next` from picking a track by hand, both end the current file with reason `stop`. So `next` (keys, the button, `wimamp next`) checks the queue itself and plays its head, falling back to `playlist-next`.

**After the queue.** mpv continues from the last played entry, which after a queue is the last queued track. Winamp behaves the same way. Returning to the track before the queue would need more state and isn't what Winamp users expect.

**The queue view reuses the playlist panel.** The window's `view()` already swaps the rows for search matches; it now returns the selection and scroll position of whichever list is shown, so navigation, mouse, scrollbar and drawing work for the queue unchanged. In the view, `remove` and `queue` turn into unqueue, so `dd` never deletes from the playlist there.

**Rejected:**
- *Move after current* (queueing reorders the playlist): no script needed, but changes the playlist order, which the user did not want.
- *A background Go process* next to mpv, listening to track changes: one language, but a second resident process with start, stop and reconnect handling on every OS.
- *The interface decides*: stops working when the TUI or window is closed.

## Risks / Trade-offs

- [A second language in the repo] → about 50 lines of Lua, using only mpv's documented script API; covered by an integration test against a real mpv.
- [Race between a queue edit and the script popping the head] → both are single property writes; the worst case is one track queued or unqueued a moment late.
- [Old mpv without entry ids or `on_before_start_file`] → both exist since mpv 0.33 (2020).
- [The queue is lost when the player quits] → like Winamp's queue; `wimamp save` saves the playlist, not the queue.

## Migration Plan

None. A running player from an older wimamp gets the script on the first queue change.
