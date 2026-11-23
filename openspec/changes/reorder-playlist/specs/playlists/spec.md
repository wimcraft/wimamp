## ADDED Requirements

### Requirement: Reorder the playlist
The TUI and the popup window SHALL let the user move the selected playlist entry one place later or earlier. The selection SHALL follow the moved entry. Moving the first entry earlier or the last entry later SHALL change nothing. Moving an entry SHALL NOT interrupt playback, SHALL keep the entry's place in the queue, and the new order SHALL be the order saved with the playlist and the order playback follows.

#### Scenario: Move down
- **WHEN** the playlist is A, B, C, B is selected and the user presses `J`
- **THEN** the playlist is A, C, B and B is still selected

#### Scenario: Playing entry moves
- **WHEN** the playing entry is moved
- **THEN** it keeps playing, and the next track is the one now after it

#### Scenario: Edge
- **WHEN** the first entry is selected and the user presses `K`
- **THEN** nothing changes
