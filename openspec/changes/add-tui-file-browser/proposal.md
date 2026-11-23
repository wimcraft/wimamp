## Why

Adding music in the TUI goes through fzf, which is a fuzzy list of every path under the home folder: fine when you know a name, awkward when you want to look around the music library the way an open dialog lets you. The user wants something closer to the window's open dialog, inside wimamp.

## What Changes

- `a` / `A` in the TUI show a file browser in the playlist panel, in place of the playlist, the way `E` shows the queue: folders and audio files of one folder, opened and left with vim/arrow keys, with marks for adding several items.
- Adding a folder adds its audio files recursively, as `wimamp add` does. The browser closes after adding; esc closes it without changes.
- The browser starts in the folder it was last left in (kept in the player while it runs), else the playing track's folder, else the home folder.
- **BREAKING (TUI only)**: fzf is no longer used; `tui_pick.go` goes away. The popup window keeps its native picker.

## Capabilities

### New Capabilities

### Modified Capabilities
- `tui`: the fzf picker is replaced by a file browser in the playlist panel.
- `keybindings`: `a` / `A` open the browser in the TUI.

## Impact

`tui_pick.go` (removed), a new `tui_browse.go`, `tui.go`, `tui_view.go`, `help.go`, README. No new dependency, fzf no longer needed. Archive after `add-tui-fzf-picker` and `reorder-playlist` (all three change the bindings table; this one carries the `J` / `K` row from `reorder-playlist`).
