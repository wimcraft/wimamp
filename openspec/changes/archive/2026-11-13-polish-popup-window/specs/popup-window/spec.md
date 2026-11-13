## MODIFIED Requirements

### Requirement: Layout
The window SHALL have a main panel and a playlist panel, in the classic Winamp style, using original artwork with no Winamp assets or branding. The main panel SHALL show elapsed time in large seven-segment digits, the scrolling title of the playing track, prev/play/pause/stop/next buttons, a seek bar, a volume slider, and shuffle and repeat toggles. The playlist panel SHALL show numbered entries with the playing one marked, the selected one highlighted, a scrollbar, and ADD and REM buttons below. There SHALL be no equalizer, visualizer, album art or lyrics. The volume percentage SHALL be shown above the volume slider, whose track SHALL be colored by level, from green at low volume through yellow to red at full volume. While the volume is being changed, the title area SHALL show the volume percentage instead of the title for about a second.

#### Scenario: Open while playing
- **WHEN** a track is playing and the user runs `wimamp gui`
- **THEN** the window shows the track title, an advancing elapsed time, the seek position, the volume, and the playlist with that track marked

#### Scenario: Change volume
- **WHEN** the user drags the volume slider or presses a volume key
- **THEN** the title area shows the new volume percentage, and the title returns shortly after the change stops

### Requirement: Mouse controls
The window SHALL support clicking the transport buttons and toggles, clicking or dragging the seek bar and volume slider, double-clicking a playlist entry to play it, scrolling the playlist with the wheel or by dragging its scrollbar, and dragging the window by its title bar. A button SHALL look pressed while the mouse button is held on it and SHALL return to its normal look when released.

#### Scenario: Seek by drag
- **WHEN** the user drags the seek bar to its midpoint and releases
- **THEN** playback jumps to half the track's length

#### Scenario: Press feedback
- **WHEN** the user presses and holds the mouse on the stop button
- **THEN** the button looks pressed until the mouse is released

### Requirement: Shuffle and repeat
The shuffle toggle SHALL shuffle and unshuffle the playlist order. The repeat toggle SHALL loop the whole playlist. Each toggle SHALL show its current state with an indicator light on the button.

#### Scenario: Repeat on
- **WHEN** repeat is on and the last entry finishes
- **THEN** playback continues with the first entry

### Requirement: Spotlight-style behavior
`wimamp gui` SHALL open the window with keyboard focus. The window SHALL close on Esc, on the close key, or when it loses focus, unless it is pinned on top. Closing the window SHALL leave playback running. At most one window SHALL exist. On macOS, closing SHALL only hide the window, and the window process SHALL stay running with no Dock icon or Cmd-Tab entry, so that the next `wimamp gui` shows it again in a fraction of the time a fresh start takes. `wimamp quit` SHALL also end the window process.

#### Scenario: Focus lost
- **WHEN** the window is open, not pinned, and the user clicks another application
- **THEN** the window closes and the music keeps playing

#### Scenario: Second invocation while open
- **WHEN** the window is open and the hotkey runs `wimamp gui` again
- **THEN** no second window appears

#### Scenario: Fast reopen
- **WHEN** the window was opened once and closed, and the user runs `wimamp gui` again
- **THEN** the window is visible again within about 200 ms, with its selection on the playing track and any search or help closed

## ADDED Requirements

### Requirement: Always on top
A pin button in the title bar SHALL toggle keeping the window above other windows, with an indicator light showing whether it is on. While pinned, the window SHALL stay open when it loses focus. The setting SHALL last until the window process ends.

#### Scenario: Pinned window loses focus
- **WHEN** the window is pinned and the user clicks another application
- **THEN** the window stays open and above that application
