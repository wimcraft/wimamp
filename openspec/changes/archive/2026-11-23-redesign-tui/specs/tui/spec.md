## MODIFIED Requirements

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
