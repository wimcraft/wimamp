## Why

Testing the popup window side by side with classic Winamp showed it looked flat and unfinished: the clock's colon rendered as two `+` signs, buttons had no press feedback, the volume slider gave no sense of level, and the seek handle was hard to find. Clicking transport buttons on an empty playlist also flashed raw mpv errors. Classic skins can't be loaded or shipped (copyrighted art, and a skin file would break the one-binary install), so the look is improved with original drawing only.

## What Changes

- Clock drawn as 7-segment digits; classic-inspired title bars, beveled buttons and a recessed seek groove with a wide handle.
- Buttons look pressed only while the mouse is held on them. Shuffle and repeat show their state with an indicator light, not a pressed button.
- The volume track is colored by level (green, yellow, red) and the percentage is shown above the slider and, briefly, in place of the title while the volume changes.
- A pin button in the title bar keeps the window above other windows; while pinned, losing focus does not close it.
- The playlist has a draggable scrollbar.
- Track actions (play, resume, pause toggle, remove, next, prev, seek) do nothing on an empty playlist instead of reporting a player error, in both the window and the TUI.

## Capabilities

### Modified Capabilities
- `popup-window`: layout details, press feedback, scrollbar, volume readout, toggle indicators, always on top, and the focus-loss exception while pinned.
- `keybindings`: track actions are ignored on an empty playlist.

## Impact

- `gui.go`, `actions.go`, `tui.go`; tests in `gui_test.go`. The bundled bitmap font is replaced by Go's own font from `golang.org/x/image`, already a dependency; no new modules.
