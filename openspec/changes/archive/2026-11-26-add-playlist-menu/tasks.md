## 1. Window

- [x] 1.1 LIST button at the right end, after the status box, which is shortened; NEW / SAVE / LOAD menu above it, drawn and closed like ADD's
- [x] 1.2 Actions `new-list`, `save-list`, `load-list` on `ctrl+n` / `ctrl+s` / `ctrl+o`; NEW runs clear
- [x] 1.3 Save / load dialogs via zenity in `<config-dir>/playlists` (created if missing), `.m3u` filter, `.m3u` added on save; guarded like the ADD picker
- [x] 1.4 Errors in the status box; cancel is a no-op
- [x] 1.5 Help and README
- [x] 1.6 macOS: ADD / SAVE / LOAD dialogs as in-process NSOpenPanel / NSSavePanel
- [x] 1.7 TUI: `ctrl+n` clears, `ctrl+s` name prompt, `ctrl+o` browser on the playlists folder

## 2. Tests

- [x] 2.1 Menu hit areas and the actions' player calls (NEW clears; save / load given a path round-trip against a private mpv)
- [x] 2.2 Name handling: `.m3u` added only when missing
- [x] 2.3 TUI keys against a private mpv; macOS string/URL bridging

## 3. Check

- [x] 3.1 Check by hand on macOS: menu, dialogs, cancel, keys (Windows not yet)
