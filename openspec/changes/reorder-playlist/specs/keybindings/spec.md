## MODIFIED Requirements

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
| add files / add folder (native picker in the window, fzf in the TUI) | `a` / `A` |
| always on top on/off (window only) | `ctrl+a` |
| queue / unqueue selected | `e` |
| queue view on/off | `E` |
| move the selected entry later / earlier (in the queue view: within the queue) | `J` / `K` |
| search / jump to track | `/` `ctrl+j` |
| queue the highlighted match, keep searching (in search) | `ctrl+e` |
| key help | `?` |
| close the interface | `q` |
| quit the player | `Q` |

#### Scenario: Play selected while paused
- **WHEN** playback is paused and the user selects another entry and presses `enter`
- **THEN** that entry plays (it does not stay paused)

#### Scenario: Winamp keys
- **WHEN** the user presses `b` in the TUI or the popup window
- **THEN** playback moves to the next track

#### Scenario: Reorder in the playlist
- **WHEN** the playlist is shown and the user presses `J`
- **THEN** the selected entry moves one place down

#### Scenario: Reorder in the queue view
- **WHEN** the queue view is shown and the user presses `J`
- **THEN** the selected entry moves one place later in the queue and the playlist order does not change
