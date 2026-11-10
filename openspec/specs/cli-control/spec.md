# cli-control Specification

## Purpose
Defines the one-shot commands that control playback, so scripts, status bars and hotkey tools can drive the player without opening an interface.

## Requirements

### Requirement: Transport commands
The system SHALL provide `play`, `pause`, `toggle`, `stop`, `next` and `prev` commands that act on the running player and exit when done.

#### Scenario: Toggle while playing
- **WHEN** a track is playing and the user runs `wimamp toggle`
- **THEN** playback pauses, and running `wimamp toggle` again resumes it

#### Scenario: Resume after stop
- **WHEN** playback was stopped with `wimamp stop` and the user runs `wimamp play` or `wimamp toggle`
- **THEN** playback restarts from the playlist entry that was current before stopping

#### Scenario: Stop keeps the playlist
- **WHEN** the user runs `wimamp stop`
- **THEN** playback stops and the playlist is unchanged

#### Scenario: Play with paths
- **WHEN** the user runs `wimamp play <paths...>`
- **THEN** the playlist is replaced by those paths and playback starts from the first entry

### Requirement: Seek and volume
The system SHALL provide `seek <seconds>` (relative, may be negative) and `vol <delta>` (relative volume change).

#### Scenario: Seek forward
- **WHEN** a track is at 0:30 and the user runs `wimamp seek 10`
- **THEN** playback continues from 0:40

#### Scenario: Missing argument
- **WHEN** the user runs `wimamp seek` with no number
- **THEN** the command exits non-zero with a usage error

### Requirement: Status line
The system SHALL provide `wimamp status`, which prints one line suitable for status bars: a state icon (playing, paused or stopped), the track title, the elapsed and total time, and the volume.

#### Scenario: Playing track
- **WHEN** "Song 1" is playing at 1:05 of 3:20 with volume 80
- **THEN** `wimamp status` prints `▶ Song 1  1:05 / 3:20  vol 80%`

#### Scenario: Stopped
- **WHEN** nothing is playing
- **THEN** `wimamp status` prints `■ stopped`

### Requirement: Errors and help
Every command SHALL exit 0 on success and non-zero with a message on stderr on failure. `wimamp help` SHALL print the usage text. An unknown command SHALL fail and point to `wimamp help`.

#### Scenario: Unknown command
- **WHEN** the user runs `wimamp bogus`
- **THEN** the command exits non-zero and stderr mentions `wimamp help`
