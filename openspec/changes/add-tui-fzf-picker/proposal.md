## Why

`a` and `A` only work in the popup window, because they open the native file picker, which a terminal (and an SSH session) can't show. The TUI help listed them anyway, so they looked broken.

## What Changes

- In the TUI, `a` / `A` run `fzf` over the home folder (files / folders, multi-select) and append the choice to the playlist.
- fzf is optional: without it the keys say how to install it or to use `wimamp add`.

## Capabilities

### New Capabilities

### Modified Capabilities
- `tui`: adding music with fzf.
- `keybindings`: `a` / `A` are no longer window-only.

## Impact

`tui_pick.go` (new), `tui.go`, `help.go`, README. No new Go dependency; fzf ≥ 0.48 is an optional runtime tool.
