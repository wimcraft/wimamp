## 1. Info

- [x] 1.1 Add `github.com/dhowden/tag`; `trackInfo(path)` returns tags, length and format, or an error; URLs short-circuit
- [x] 1.2 Info view state in the TUI and the window (`i` toggles, esc closes, selection kept); read off the UI loop
- [x] 1.3 Draw it in the playlist panel (TUI and window): label/value rows, long paths wrapped or cut from the left
- [x] 1.4 `f`: reveal in Finder / Explorer / xdg-open the folder; errors in the bottom border
- [x] 1.5 Help and README

## 2. Tests

- [x] 2.1 `trackInfo` on small tagged and untagged fixtures (MP3, FLAC), missing file, URL
- [x] 2.2 Reveal command per OS (built, not run)
- [x] 2.3 TUI view golden and exact widths

## 3. Check

- [ ] 3.1 Check by hand: TUI and window, tagged and untagged files, `f` on macOS and Windows
