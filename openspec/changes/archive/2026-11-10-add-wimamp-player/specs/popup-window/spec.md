## Purpose

Defines `wimamp gui`, a small Winamp-style window that a hotkey can summon to change tracks without going back to the terminal.

## ADDED Requirements

### Requirement: Layout
The window SHALL have a main panel and a playlist panel, in the classic Winamp style, using original artwork with no Winamp assets or branding. The main panel SHALL show elapsed time in large digits, the scrolling title of the playing track, prev/play/pause/stop/next buttons, a seek bar, a volume slider, and shuffle and repeat toggles. The playlist panel SHALL show numbered entries with the playing one marked, the selected one highlighted, and ADD and REM buttons below. There SHALL be no equalizer, visualizer, album art or lyrics.

#### Scenario: Open while playing
- **WHEN** a track is playing and the user runs `wimamp gui`
- **THEN** the window shows the track title, an advancing elapsed time, the seek position, the volume, and the playlist with that track marked

### Requirement: Keyboard first
The window SHALL accept the same keybindings as the TUI, including user overrides, and SHALL be fully usable without a mouse except for choosing files in the ADD dialog.

#### Scenario: Play from the window by keyboard
- **WHEN** the window is open and the user presses `j` `j` `enter`
- **THEN** the entry two below the selection starts playing

### Requirement: Mouse controls
The window SHALL support clicking the transport buttons and toggles, clicking or dragging the seek bar and volume slider, double-clicking a playlist entry to play it, scrolling the playlist with the wheel, and dragging the window by its title bar.

#### Scenario: Seek by drag
- **WHEN** the user drags the seek bar to its midpoint and releases
- **THEN** playback jumps to half the track's length

### Requirement: Shuffle and repeat
The shuffle toggle SHALL shuffle and unshuffle the playlist order. The repeat toggle SHALL loop the whole playlist. Each toggle SHALL show its current state.

#### Scenario: Repeat on
- **WHEN** repeat is on and the last entry finishes
- **THEN** playback continues with the first entry

### Requirement: Add and remove entries
ADD SHALL open the operating system's native picker for choosing files or a folder, and append the chosen audio to the playlist using the same rules as `wimamp add`. REM SHALL remove the selected entry. Cancelling the picker SHALL change nothing.

#### Scenario: Add a folder
- **WHEN** the user clicks ADD and picks a folder of mp3 files
- **THEN** those files are appended to the playlist and appear in the window

### Requirement: Spotlight-style behavior
`wimamp gui` SHALL open the window with keyboard focus. The window SHALL close on Esc, on the close key, or when it loses focus. Closing the window SHALL leave playback running. At most one window SHALL exist. On macOS, closing SHALL only hide the window, and the window process SHALL stay running with no Dock icon or Cmd-Tab entry, so that the next `wimamp gui` shows it again in a fraction of the time a fresh start takes. `wimamp quit` SHALL also end the window process.

#### Scenario: Focus lost
- **WHEN** the window is open and the user clicks another application
- **THEN** the window closes and the music keeps playing

#### Scenario: Second invocation while open
- **WHEN** the window is open and the hotkey runs `wimamp gui` again
- **THEN** no second window appears

#### Scenario: Fast reopen
- **WHEN** the window was opened once and closed, and the user runs `wimamp gui` again
- **THEN** the window is visible again within about 200 ms, with its selection on the playing track and any search or help closed

### Requirement: Low idle cost
While open, the window SHALL redraw only on input, on a status change, or to scroll a title that doesn't fit (at most about five times per second), not continuously. Measured with `top` over a few seconds on a paused, idle window, its CPU use SHALL stay under 2% of one core.

#### Scenario: Idle window
- **WHEN** the window is open and paused with no input
- **THEN** its CPU use stays near idle

#### Scenario: Hidden window
- **WHEN** the window has been closed but its process stays resident
- **THEN** it does not poll the player, and its CPU use measures as 0.0% in `top`
