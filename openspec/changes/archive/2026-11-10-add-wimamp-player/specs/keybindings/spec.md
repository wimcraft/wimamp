## Purpose

Defines the keyboard controls shared by the TUI and the popup window, and how users change them.

## ADDED Requirements

### Requirement: Default bindings
The system SHALL provide these default bindings in both the TUI and the popup window. `z x c v b` follow the classic Winamp layout (prev, play, pause, stop, next).

| Action | Keys |
|---|---|
| down / up | `j` `down` / `k` `up` |
| top / bottom | `gg` `home` / `G` `end` |
| half page down / up | `ctrl+d` `pgdown` / `ctrl+u` `pgup` |
| jump to current track | `o` |
| play selected | `enter` |
| resume / play | `x` |
| toggle pause | `space` `c` |
| stop | `s` `v` |
| next / prev | `n` `b` / `p` `z` |
| seek +5 s / -5 s | `l` `right` / `h` `left` |
| volume up / down | `+` `=` / `-` |
| remove selected | `dd` |
| shuffle / repeat toggle | `S` / `r` |
| add files / add folder (window only, opens the native picker) | `a` / `A` |
| search / jump to track | `/` `ctrl+j` |
| key help | `?` |
| close the interface | `q` |
| quit the player | `Q` |

#### Scenario: Play selected while paused
- **WHEN** playback is paused and the user selects another entry and presses `enter`
- **THEN** that entry plays (it does not stay paused)

#### Scenario: Winamp keys
- **WHEN** the user presses `b` in the TUI or the popup window
- **THEN** playback moves to the next track

### Requirement: Vim-style sequences
The system SHALL support multi-key sequences such as `gg` and `dd`. A key that starts a sequence SHALL wait for the next key. If the next key does not continue any sequence, that key SHALL be handled on its own.

#### Scenario: gg
- **WHEN** the user presses `g` then `g`
- **THEN** the selection moves to the first entry

#### Scenario: Broken sequence
- **WHEN** the user presses `g` then `G`
- **THEN** the selection moves to the last entry

### Requirement: User overrides
The system SHALL read `<config-dir>/keys.conf`, which has lines of the form `action = key key ...` and `#` comments. A listed action SHALL get exactly the given keys, replacing its defaults. An action with no keys after `=` SHALL be unbound. A key the user binds SHALL win over any default binding of the same key. A missing file SHALL mean the defaults.

#### Scenario: Rebind next
- **WHEN** keys.conf contains `next = J`
- **THEN** `J` plays the next track, `n` no longer does, and all other defaults still work

#### Scenario: Unknown action
- **WHEN** keys.conf contains `bogus = x`
- **THEN** the TUI and the popup window refuse to start, with an error that names the line number

### Requirement: Key help
The system SHALL show every action with a short description and its current keys, including user overrides, when `?` is pressed in the TUI or the window. Any key SHALL close the help. `wimamp keys` SHALL print the same list.

#### Scenario: Help reflects overrides
- **WHEN** keys.conf contains `next = J` and the user presses `?`
- **THEN** the help lists `J` for the next track, and not `n`

### Requirement: Terminal control-key equivalents in the window
The window SHALL treat `ctrl+m` as `enter`, `ctrl+i` as `tab`, `ctrl+h` as `backspace` and `ctrl+[` as Esc, as terminals do, so the same muscle memory works in the TUI and the window.

#### Scenario: ctrl+m plays
- **WHEN** the window is open and the user presses `ctrl+m`
- **THEN** the selected entry plays, as with `enter`
