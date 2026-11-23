# tui Specification

## Purpose
Defines the terminal interface, opened by running `wimamp` with no arguments or `wimamp tui`, for browsing the playlist and controlling playback from the keyboard.

## Requirements

### Requirement: Layout
The TUI SHALL draw two boxes across the terminal width, in the colors of the popup window. The player box SHALL show the play state, the elapsed and total time, the playing entry's number and title, a seek bar, the volume as a meter colored by level with its percentage, and whether shuffle and repeat are on. The playlist box below SHALL fill the remaining height with the numbered, scrollable playlist; its top border SHALL show the list's title and the selected position with the playlist's total length, and its bottom border SHALL show key hints, or the search prompt while searching, or the last command error. The playing entry SHALL be marked and drawn in white, and the selected entry highlighted with a bar across the box. An empty playlist SHALL show a hint on how to add music. Every line SHALL fit the terminal width, counting wide characters as two columns.

#### Scenario: Empty playlist
- **WHEN** the TUI opens with an empty playlist
- **THEN** it shows a hint mentioning `wimamp add`

#### Scenario: Wide characters
- **WHEN** the playlist holds a file named in Japanese
- **THEN** its length is right-aligned with the other rows and the box border stays straight

### Requirement: Live state
The TUI SHALL reflect changes made from other interfaces within one second.

#### Scenario: External skip
- **WHEN** the TUI is open and the user runs `wimamp next` from another terminal
- **THEN** within one second the TUI marks the new entry as playing and shows it in the player box

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

### Requirement: File browser
In the TUI, `a` and `A` SHALL show a file browser in the playlist panel in place of the playlist. It SHALL list the folders and then the audio files of one folder, each group sorted by name ignoring case, without hidden entries, and SHALL show that folder's path in the panel title. The user SHALL be able to move the selection with the usual navigation keys, open the selected folder, go to the parent folder with the folder just left selected, and go to the home folder. The user SHALL be able to mark and unmark several files and folders, also across folders. Adding SHALL append the marked items in the order marked, or the selected item when none is marked, to the playlist as `wimamp add` does, folders recursively, and SHALL close the browser. Esc SHALL close the browser without changes. The bottom border SHALL show how many items are marked. The browser SHALL open in the folder it was last left in while the player runs, otherwise in the playing track's folder, otherwise in the home folder. A folder that cannot be read SHALL show its error in the bottom border and leave the browser in the current folder. Player controls other than navigation and adding SHALL NOT act while the browser is open, except help and closing the TUI.

#### Scenario: Open a folder and add a file
- **WHEN** the user presses `a`, opens the folder `Albums`, selects `01 Intro.mp3` and presses enter
- **THEN** the browser closes and `01 Intro.mp3` is appended to the playlist

#### Scenario: Add several
- **WHEN** the user marks folder `A`, goes to another folder, marks file `x.flac` and presses `a`
- **THEN** the audio files under `A` and then `x.flac` are appended, in that order

#### Scenario: Go up
- **WHEN** the browser shows `~/Music/Albums` and the user presses `h`
- **THEN** it shows `~/Music` with `Albums/` selected

#### Scenario: Remembers the folder
- **WHEN** the user closes the browser in `~/Music/Jazz`, quits the TUI, starts it again and presses `a`
- **THEN** the browser opens in `~/Music/Jazz`

#### Scenario: Cancel
- **WHEN** the user marks two files and presses esc
- **THEN** the playlist is shown again, unchanged

#### Scenario: Only music
- **WHEN** a folder holds `cover.jpg`, `.DS_Store`, `song.mp3` and a subfolder `CD2`
- **THEN** the browser lists `CD2/` and `song.mp3`
