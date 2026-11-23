## Why

When a track catches your ear there is no way to find out what it is: the playlist shows only a title or a file name, with no artist, album or location.

## What Changes

- `i` shows the selected entry's info in the playlist panel, the way `E` shows the queue, in the TUI and the popup window: title, artist, album, year, track number, genre, length, format, and the full path.
- Tags are read from the file itself, so any entry works, not only the playing one.
- `f` in the info view opens the file manager with the file selected (Finder, Explorer; on Linux the folder).

## Capabilities

### New Capabilities
- `track-info`: the info view and show-in-folder.

### Modified Capabilities
- `keybindings`: `i` and `f`.

## Impact

New `info.go`, `tui.go`, `tui_view.go`, `gui.go`, `help.go`, README. New Go dependency `github.com/dhowden/tag` (pure Go, BSD-2, no cgo) for ID3/MP4/FLAC/OGG tags. Archive after `add-tui-file-browser` (each restates the bindings table).
