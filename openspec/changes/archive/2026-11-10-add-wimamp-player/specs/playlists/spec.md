## Purpose

Defines how music gets into the play queue and how queues are kept as named playlists, using plain files and folders without a media library.

## ADDED Requirements

### Requirement: Add files and folders
The system SHALL let the user append files and folders to the current playlist with `wimamp add <paths...>`. Folders SHALL be searched recursively for audio files (mp3, flac, ogg, oga, opus, m4a, mp4, aac, wav, aif, aiff, wma, ape, wv, mka, webm, alac) in lexical path order. Files given explicitly SHALL be added whatever their extension. Relative paths SHALL resolve against the caller's working directory.

#### Scenario: Add a folder
- **WHEN** the user runs `wimamp add ./album` from a directory containing `album/01.mp3`, `album/02.flac` and `album/cover.jpg`
- **THEN** the two audio files are appended in order and `cover.jpg` is skipped

#### Scenario: Nothing to add
- **WHEN** the given folder contains no audio files
- **THEN** the command exits non-zero with "no audio files found" and the playlist is unchanged

#### Scenario: Path does not exist
- **WHEN** a given path does not exist
- **THEN** the command exits non-zero with an error naming the path

### Requirement: Clear
The system SHALL empty the playlist and stop playback on `wimamp clear`.

#### Scenario: Clear
- **WHEN** the user runs `wimamp clear`
- **THEN** the playlist is empty and nothing is playing

### Requirement: Save, load and list playlists
The system SHALL save the current playlist as an `.m3u` file with `wimamp save <name>`, replace the current playlist with a saved one and start playing it with `wimamp load <name>`, and print saved playlist names with `wimamp playlists`. A bare name SHALL refer to `<config-dir>/playlists/<name>.m3u`. A name containing a path separator or ending in `.m3u` SHALL be used as a file path. Saved files SHALL contain absolute paths, so they work from any directory.

#### Scenario: Round trip
- **WHEN** the user runs `wimamp save evening`, then `wimamp clear`, then `wimamp load evening`
- **THEN** the playlist is restored in the same order and playback starts

#### Scenario: Load a missing playlist
- **WHEN** the user runs `wimamp load nope` and no such playlist exists
- **THEN** the command exits non-zero and the current playlist is unchanged

### Requirement: Entry names
Playlist entries SHALL be displayed by their file name without its extension. The track that is playing SHALL be displayed by the title the playback engine reports, when there is one.

#### Scenario: Unplayed entry
- **WHEN** `/music/03 - Echo.mp3` is in the playlist but has not been played
- **THEN** it is listed as `03 - Echo`
