## REMOVED Requirements

### Requirement: Add music with fzf
**Reason**: replaced by the file browser, which shows the folder structure.
**Migration**: press `a` or `A` as before; the browser opens in the playlist panel. `wimamp add <path>` is unchanged.

## ADDED Requirements

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
