## Purpose

Defines the terminal interface, opened by running `wimamp` with no arguments or `wimamp tui`, for browsing the playlist and controlling playback from the keyboard.

## ADDED Requirements

### Requirement: Layout
The TUI SHALL show a status line (in the same format as `wimamp status`) above a numbered, scrollable playlist. The playing entry SHALL be marked and the selected entry highlighted. An empty playlist SHALL show a hint on how to add music.

#### Scenario: Empty playlist
- **WHEN** the TUI opens with an empty playlist
- **THEN** it shows a hint mentioning `wimamp add`

### Requirement: Live state
The TUI SHALL reflect changes made from other interfaces within one second.

#### Scenario: External skip
- **WHEN** the TUI is open and the user runs `wimamp next` from another terminal
- **THEN** within one second the TUI marks the new entry as playing and updates the status line

### Requirement: Initial selection
When the TUI opens, the selection SHALL be on the playing entry, or on the first entry if nothing is playing.

#### Scenario: Open mid-playlist
- **WHEN** entry 12 is playing and the user opens the TUI
- **THEN** entry 12 is selected and visible

### Requirement: Closing keeps playing
Closing the TUI with the close key or `ctrl+c` SHALL leave playback running. Only the quit-player action SHALL stop the player.

#### Scenario: Close TUI
- **WHEN** music is playing and the user presses `q`
- **THEN** the TUI exits and the music keeps playing

### Requirement: Command errors are shown
When a player command fails (e.g. playing an unreadable file), the TUI SHALL show the error until the next key press, without exiting.

#### Scenario: Engine error
- **WHEN** a command sent from the TUI is rejected by the player
- **THEN** an error message appears at the bottom and the TUI stays open
