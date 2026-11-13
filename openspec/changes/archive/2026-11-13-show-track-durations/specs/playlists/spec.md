## ADDED Requirements

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
