## ADDED Requirements

### Requirement: Add music with fzf
In the TUI, `a` and `A` SHALL hand the terminal to `fzf` to choose files or folders under the home folder, several at once, and append the chosen entries to the playlist as `wimamp add` does. The TUI SHALL come back when fzf closes; cancelling SHALL change nothing. A user's `FZF_DEFAULT_COMMAND` SHALL NOT replace the folder walk. When `fzf` is not installed, the TUI SHALL say so in its bottom border and point to `wimamp add`.

#### Scenario: Add a folder
- **WHEN** the user presses `A`, types part of a folder name and presses enter
- **THEN** the TUI returns and the folder's audio files are appended to the playlist

#### Scenario: Cancel
- **WHEN** the user presses `a` and then esc in fzf
- **THEN** the TUI returns with the playlist unchanged

#### Scenario: fzf missing
- **WHEN** `fzf` is not on the PATH and the user presses `a`
- **THEN** the bottom border says to install fzf or use `wimamp add`
