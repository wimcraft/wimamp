## Why

In a big music folder the TUI file browser can only be scrolled; there is no quick way to jump to an artist, unlike the playlist, where `/` searches.

## What Changes

- `/` in the file browser filters the shown folder as you type (names containing the text, case ignored); enter opens or adds the selected match, tab marks it, esc clears the search, a second esc closes the browser.

## Capabilities

### New Capabilities

### Modified Capabilities
- `tui`: search in the file browser.

## Impact

`tui_browse.go` and its test. No new dependency.
