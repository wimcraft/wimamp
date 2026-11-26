## ADDED Requirements

### Requirement: Playlist menu
The playlist panel SHALL have a LIST button at the right end of its bottom bar, after the status box, that opens a menu with NEW, SAVE and LOAD, closed again by choosing an item or clicking elsewhere. NEW SHALL empty the playlist and stop playback as `wimamp clear` does. SAVE SHALL open the operating system's native save dialog in the saved-playlists folder and save the playlist there as `wimamp save` does, adding `.m3u` to a name without an extension. LOAD SHALL open the native open dialog in the saved-playlists folder for `.m3u` and `.m3u8` files and replace the playlist with the chosen one and start playing it, as `wimamp load` does. Cancelling a dialog SHALL change nothing. A failure SHALL be shown in the status box.

#### Scenario: New playlist
- **WHEN** the user clicks LIST and then NEW
- **THEN** the playlist is empty and nothing is playing

#### Scenario: Save and load
- **WHEN** the user clicks LIST, SAVE, types `evening` and confirms, then clicks LIST, NEW, then LIST, LOAD and picks `evening.m3u`
- **THEN** the playlist is restored in the same order and playback starts, and `wimamp playlists` lists `evening`

#### Scenario: Cancel
- **WHEN** the user clicks LIST, LOAD and cancels the dialog
- **THEN** the playlist and playback are unchanged

#### Scenario: Menu closes
- **WHEN** the LIST menu is open and the user clicks the playlist
- **THEN** the menu closes and nothing else changes
