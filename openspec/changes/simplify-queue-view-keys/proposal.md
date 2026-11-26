## Why

In the queue view `e` unqueued the selected entry. The user reaches for `e` (or `E`) to go back to the playlist, so it kept removing tracks from the queue by accident.

## What Changes

- In the queue view, `e` does nothing; a single `d` unqueues the selected entry (instead of `dd` or `e`). `E` and Esc still go back.

## Capabilities

### New Capabilities

### Modified Capabilities
- `playlists`: queue view keys.

## Impact

`actions.go`, `tui.go`, `gui.go`, `tui_view.go`, README. A habitual `dd` in the queue view now unqueues two entries.
