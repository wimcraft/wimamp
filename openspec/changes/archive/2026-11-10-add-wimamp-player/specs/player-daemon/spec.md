## Purpose

Defines the background player process that owns playback, so every interface (CLI, TUI, popup window, hotkey tools) controls the same player state.

## ADDED Requirements

### Requirement: Start on demand
The system SHALL start the background player automatically the first time any command that needs it is run, without the user starting anything separately.

#### Scenario: First command starts the player
- **WHEN** no player is running and the user runs `wimamp play ~/Music`
- **THEN** a background player starts, the command succeeds, and playback begins

#### Scenario: Player engine missing
- **WHEN** the playback engine is not installed or not on `PATH`
- **THEN** the command exits with a non-zero status and an error message that names the missing engine

### Requirement: Single player instance
The system SHALL run at most one background player per user, including when commands run concurrently or a previous player left a stale socket behind.

#### Scenario: Concurrent first commands
- **WHEN** two `wimamp` commands start at the same moment and no player is running
- **THEN** exactly one background player is started and both commands talk to it

#### Scenario: Stale socket
- **WHEN** a socket file exists but no player answers on it
- **THEN** the system replaces it with exactly one new player and the command succeeds

#### Scenario: Player fails to come up
- **WHEN** a started player does not answer within a few seconds
- **THEN** the command exits non-zero with an error, and repeating the command does not leave more than one player process running

### Requirement: Survive the launching terminal
The background player SHALL keep playing when the terminal, TUI or window that started it is closed.

#### Scenario: Terminal closed
- **WHEN** the user starts playback from a terminal and closes that terminal window
- **THEN** playback continues and other terminals can still control it

### Requirement: Shared control endpoint
The player SHALL be reachable at one well-known, per-user endpoint that resolves the same way from terminals, launchers and hotkey daemons. Setting the `WIMAMP_SOCKET` environment variable SHALL override it.

#### Scenario: Control from a launcher
- **WHEN** playback was started from a terminal and the user runs `wimamp toggle` from Raycast, Hammerspoon, i3 or AutoHotkey
- **THEN** the same player toggles pause

### Requirement: Player death while an interface is open
If the player stops (it crashed, or `wimamp quit` ran elsewhere) while the TUI or the window is open, the interface SHALL show "player not running" instead of stale or zeroed playback state, and SHALL NOT restart the player by itself. The next CLI command, `wimamp tui`, or `wimamp gui` SHALL start a new player with an empty playlist.

#### Scenario: Player killed under the TUI
- **WHEN** the TUI is open and the player process is killed
- **THEN** within one second the TUI shows "player not running", and running `wimamp add ~/Music` starts a new player

### Requirement: Quit
The system SHALL stop the background player on `wimamp quit`. Quitting when no player is running SHALL succeed without starting one.

#### Scenario: Quit when idle
- **WHEN** no player is running and the user runs `wimamp quit`
- **THEN** the command exits 0 and no player process is started
