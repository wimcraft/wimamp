## Why

The window has a classic Winamp look, but the TUI is a plain status line over a list. The TUI is how wimamp is used from a terminal, so it should feel like the same player while staying minimal.

## What Changes

- The TUI is drawn as two rounded boxes in the window's palette: a player box (play state, time, title, a seek bar, a volume meter colored by level, shuffle and repeat lights) above a playlist box.
- The playlist box carries its title (`PLAYLIST`, `QUEUE (n)` in the queue view, `KEYS` for help) and the position and total length in its top border, and key hints, the search prompt or the last error in its bottom border.
- Rows keep their content (number, queue mark, name, length); the playing row is white and bold, the selection a blue bar across the box. Wide characters are measured as two columns, so names in Japanese or Chinese no longer push lengths out of line.
- Keys, search, the queue and all behavior stay the same. `wimamp status` keeps its one-line format.

## Capabilities

### New Capabilities

### Modified Capabilities
- `tui`: the layout replaces the status line with a player panel and boxes the playlist.

## Impact

- New `tui_view.go` with the drawing; `tui.go` keeps the model and key handling.
- lipgloss and charmbracelet/x/ansi, already in the module graph through bubbletea, become direct dependencies; nothing new is downloaded.
