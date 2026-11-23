## 1. Browser

- [x] 1.1 Folder listing: folders then audio files, case-insensitive sort, hidden skipped, symlinks followed, read errors returned
- [x] 1.2 Browser state in the TUI model: folder, selection, scroll, marks in order; keys as in design.md
- [x] 1.3 Draw it in the playlist panel: folder in the title, `/` after folders, mark sign, marked count and key hints in the bottom border
- [x] 1.4 Add through `load(…, "append")` off the UI loop; close on add, esc closes
- [x] 1.5 Start folder: `user-data/wimamp/browse-dir`, else playing track's folder, else home; stored when the browser closes
- [x] 1.6 Remove the fzf picker (`tui_pick.go`, its test, README/help mentions)

## 2. Tests

- [x] 2.1 Listing on a temp folder (order, filtering, hidden, unreadable)
- [x] 2.2 Key handling: open, up with selection kept, marks across folders, add order, esc
- [x] 2.3 View golden at 50x14 and every line exactly the width

## 3. Check

- [ ] 3.1 Check by hand on macOS and Windows terminals
