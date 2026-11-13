## ADDED Requirements

### Requirement: Empty playlist
Actions that need a track (play selected, resume, toggle pause, remove, next, prev, seek) SHALL do nothing when the playlist is empty, in both the TUI and the popup window, instead of reporting a player error. Stop, volume, shuffle and repeat SHALL keep working.

#### Scenario: Remove on an empty playlist
- **WHEN** the playlist is empty and the user clicks REM or presses the remove key
- **THEN** nothing happens and no error is shown
