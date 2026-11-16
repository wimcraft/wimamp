# playlists Specification

## Purpose
Defines how music gets into the play queue and how queues are kept as named playlists, using plain files and folders without a media library.

## Requirements

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

### Requirement: Entry durations
The TUI and the popup window SHALL show the length of each playlist entry next to its name, in the same `m:ss` format used for the playing track. Lengths SHALL be read from the audio file itself for MP3, FLAC, Ogg Vorbis, Opus, M4A/MP4 (including ALAC), WAV and AIFF files. An entry whose length cannot be read (another format, a missing or damaged file, or a stream URL) SHALL show no length rather than an error or a placeholder, and SHALL still play normally. The playing entry SHALL show the length the playback engine reports, and that length SHALL then be shown for the same file for as long as the interface runs, replacing a length read from the file or filling in one that could not be read. Showing a playlist SHALL NOT wait for lengths to be read: entries appear at once and their lengths fill in as they become known, visible entries first. Lengths SHALL NOT be stored on disk.

#### Scenario: Unplayed entries
- **WHEN** a playlist holds `/music/a.mp3` (3 min 15 s) and `/music/b.flac` (4 min 2 s), and neither has been played
- **THEN** the playlist lists them with `3:15` and `4:02`

#### Scenario: Unreadable entry
- **WHEN** the playlist holds a `.wma` file, or a file that has been deleted since it was added
- **THEN** that entry shows its name with no length, and no error is shown

#### Scenario: Playing entry
- **WHEN** an entry is playing
- **THEN** its length is the one the playback engine reports, even for a format wimamp cannot read

#### Scenario: Length learned from playback
- **WHEN** a `.wma` entry with no shown length plays to its end and the next track starts
- **THEN** the `.wma` entry keeps showing the length the playback engine reported

#### Scenario: Large playlist
- **WHEN** the user opens the window or TUI with a 500-entry playlist
- **THEN** the entries appear immediately, and lengths for the visible entries appear shortly after

### Requirement: Playlist total length
The popup window and the TUI SHALL show the total length of the playlist: the sum of all known entry lengths, as `m:ss` below one hour and `h:mm:ss` from one hour on. While any entry's length is unknown, the total SHALL be marked with a trailing `+`. The total SHALL update as lengths become known and as entries are added or removed. An empty playlist SHALL show no total.

#### Scenario: All lengths known
- **WHEN** the playlist holds three entries of 1:00, 2:30 and 0:45
- **THEN** the total shows `4:15`

#### Scenario: Some lengths unknown
- **WHEN** the playlist holds entries totalling 1 h 2 min 5 s that can be read, plus one `.wma` entry that has not been played
- **THEN** the total shows `1:02:05+`

### Requirement: Play queue
The user SHALL be able to queue playlist entries to play next. Queueing an entry SHALL add it to the end of the queue; queueing an entry that is already queued SHALL remove it from the queue. When a track ends on its own, or on `next`, the first queued entry still in the playlist SHALL play and leave the queue; with the queue empty, playback SHALL continue as before. After queued entries have played, the playlist SHALL continue after the last one played, as in Winamp. A queued entry that starts playing for any other reason SHALL leave the queue, and so SHALL an entry removed from the playlist. Queueing SHALL NOT change the playlist order. The queue SHALL work with no wimamp interface open, SHALL last as long as the player runs, and SHALL NOT be saved with the playlist. The TUI and the popup window SHALL show each queued entry's place in the queue before its name, as `[1]`, `[2]` and so on.

#### Scenario: Queued tracks play first
- **WHEN** entry 1 of a four-entry playlist is playing and the user queues entry 4 and then entry 2
- **THEN** the entries play in the order 1, 4, 2, 3

#### Scenario: Queue at the end of the playlist
- **WHEN** the last entry is playing, repeat is off, and entry 1 is queued
- **THEN** entry 1 plays after the last entry ends

#### Scenario: Next takes the queue
- **WHEN** entry 3 is queued and the user presses the next key or runs `wimamp next`
- **THEN** entry 3 plays and leaves the queue

#### Scenario: Unqueue
- **WHEN** the user queues an entry that is already queued
- **THEN** it leaves the queue and stays in the playlist

#### Scenario: Marks
- **WHEN** two entries are queued
- **THEN** the first one's row shows `[1]` and the second one's `[2]` before the name, and each mark disappears when that entry starts playing

### Requirement: Queue view
The TUI and the popup window SHALL switch the playlist list to a view of the queue and back. The view SHALL list the queued entries in play order, in the same row format as the playlist. In the window the panel title SHALL read `WIMAMP QUEUE`; the TUI SHALL show a header line with the number of queued entries. An empty queue SHALL show a one-line hint. In the view, navigation and the mouse SHALL work as in the playlist; playing an entry SHALL play it; removing or queueing an entry SHALL only remove it from the queue, never from the playlist; and the user SHALL be able to move the selected entry one place later or earlier in the queue. Esc SHALL return to the playlist. Player controls SHALL work as usual in the view.

#### Scenario: Reorder the queue
- **WHEN** entries A and B are queued, in that order, and the user selects B in the queue view and moves it earlier
- **THEN** the queue view lists B before A, and B plays first

#### Scenario: Remove in the queue view
- **WHEN** the user presses the remove key on an entry in the queue view
- **THEN** the entry leaves the queue and stays in the playlist
