## Why

The window has no way to start over or switch playlists: emptying the list means `dd` on every entry or `wimamp clear` in a shell, and saved playlists are only reachable through `wimamp save` / `wimamp load`. Winamp's playlist editor has a LIST button next to ADD and REM with NEW LIST, SAVE LIST and LOAD LIST.

## What Changes

- A LIST button at the right end of the window's bottom bar, after the status box (`[ADD][REM] [status…][LIST]`), as in Winamp, opens a small menu above it, like ADD's FILE / DIR menu: NEW, SAVE, LOAD.
- NEW empties the playlist and stops playback, as `wimamp clear` does.
- SAVE opens the native save dialog in the saved-playlists folder and writes an `.m3u`, as `wimamp save` does.
- LOAD opens the native open dialog there, for `.m3u` / `.m3u8`, and replaces the playlist with it and plays, as `wimamp load` does.
- Keys `ctrl+n` / `ctrl+s` / `ctrl+o` do the same from the keyboard, as in Winamp's playlist editor, in the window and the TUI. In the TUI, save asks for a name in the bottom border and load shows the file browser on the saved-playlists folder.
- On macOS the window's file dialogs (ADD, SAVE, LOAD) run inside wimamp as native panels instead of through zenity's separate osascript process, which didn't get keyboard focus and made the window lose focus and close when it was cancelled.

## Capabilities

### New Capabilities

### Modified Capabilities
- `popup-window`: the LIST button and its menu.
- `keybindings`: `ctrl+n`, `ctrl+s`, `ctrl+o`.
- `tui`: new, save and load playlists.

## Impact

`gui.go` (button, menu, dialogs, actions), `keys.go`, `help.go`, README. Reuses the existing save / load / clear code and the zenity dialogs ADD already uses; no new dependency.
