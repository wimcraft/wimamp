## ADDED Requirements

### Requirement: New, save and load playlists
In the TUI, the new-playlist key SHALL empty the playlist and stop playback as `wimamp clear` does. The save key SHALL ask for a name in the bottom border and save the playlist as `wimamp save <name>` does; esc SHALL cancel and an empty name SHALL save nothing. The load key SHALL show the file browser on the saved-playlists folder, listing folders and `.m3u` / `.m3u8` files, without marks; enter on a playlist SHALL replace the playlist with it and start playing, as `wimamp load` does, and esc SHALL close it unchanged.

#### Scenario: Save and load in the TUI
- **WHEN** the user presses `ctrl+s`, types `eve`, presses enter, then `ctrl+n`, then `ctrl+o` and enter on `eve.m3u`
- **THEN** the playlist is restored in the same order and plays

#### Scenario: Cancel saving
- **WHEN** the user presses `ctrl+s` and then esc
- **THEN** nothing is saved
