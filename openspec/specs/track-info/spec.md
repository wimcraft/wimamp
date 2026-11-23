# track-info Specification

## Purpose
TBD - created by archiving change add-track-info. Update Purpose after archive.

## Requirements

### Requirement: Track info view
The TUI and the popup window SHALL show the selected entry's info in the playlist panel in place of the list, and switch back on the same key or esc. The view SHALL show the full path and the containing folder, the length when known, the format, and those of title, artist, album, album artist, year, track number, disc number and genre that the file's tags hold, read from the file itself so that any entry works, whether it is playing or not. Fields the file does not have SHALL be left out. Reading SHALL NOT block the interface or interrupt playback. For a URL the view SHALL show only the URL. A file that cannot be read SHALL show its path and the error. Player controls SHALL keep working while the view is shown.

#### Scenario: Info for a track that is not playing
- **WHEN** track 5 is playing and the user selects track 12, a tagged MP3, and presses `i`
- **THEN** the panel shows track 12's artist, album, year and full path, and track 5 keeps playing

#### Scenario: Untagged file
- **WHEN** the selected file has no tags
- **THEN** the view shows its path, folder, length and format only

#### Scenario: Back
- **WHEN** the info view is shown and the user presses esc
- **THEN** the playlist is shown with the same entry selected

### Requirement: Show in folder
In the info view, the user SHALL be able to open the system file manager at the track's file: with the file selected on macOS and Windows, at its folder elsewhere. A failure SHALL be shown, not ignored.

#### Scenario: Reveal
- **WHEN** the info view shows `~/Music/Album/03.flac` on macOS and the user presses `f`
- **THEN** Finder opens `~/Music/Album` with `03.flac` selected

#### Scenario: No desktop
- **WHEN** the file manager cannot be started
- **THEN** the error is shown in the bottom border
