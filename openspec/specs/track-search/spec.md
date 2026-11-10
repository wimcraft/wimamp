# track-search Specification

## Purpose
Defines jump-to-track: finding an entry in a long playlist by typing part of its name, in both the TUI and the popup window.

## Requirements

### Requirement: Search prompt
Pressing a search key (`/` or `ctrl+j` by default) SHALL open a search prompt that filters the playlist live as the user types. An entry SHALL match when its display name or its parent folder name contains the query, ignoring case. Entries that don't contain it SHALL still match when the query's characters appear in order in the name (fuzzy). Contains-matches SHALL be listed before fuzzy matches, each group in playlist order. While the prompt is open, typed characters SHALL go to the query, not to keybindings.

#### Scenario: Substring before fuzzy
- **WHEN** the playlist has "Echo", "Night Ferry - Echo" and "Each Coffee Hour" and the user types `echo`
- **THEN** "Echo" and "Night Ferry - Echo" are listed, and "Each Coffee Hour" (a fuzzy match: e-c-h-o in order) comes after them

#### Scenario: Folder match
- **WHEN** a track's folder is `Night Ferry/Static Bloom` and the user types `static bloom`
- **THEN** the tracks in that folder are listed

### Requirement: Choosing a match
In the prompt, `up`/`down` and `ctrl+p`/`ctrl+n` SHALL move the selection among the matches, `backspace` SHALL edit the query, `enter` SHALL play the selected match, close the prompt and move the playlist selection to it, and Esc SHALL close the prompt without changing playback or the selection.

#### Scenario: Jump and play
- **WHEN** the user presses `/`, types `stop`, and presses `enter`
- **THEN** the first matching track plays and is selected in the playlist

#### Scenario: Cancel
- **WHEN** the user presses `/`, types something, and presses Esc
- **THEN** the full playlist is shown again, with the previous selection and playback unchanged

#### Scenario: Esc in the window
- **WHEN** the search prompt is open in the popup window and the user presses Esc
- **THEN** only the prompt closes, not the window
