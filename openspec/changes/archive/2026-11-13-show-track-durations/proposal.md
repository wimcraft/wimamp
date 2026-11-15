## Why

The playlist shows a track's length only while it plays, because the length comes from mpv and mpv only knows it for the loaded file. Classic Winamp shows a length on every row, and scanning a playlist for a long or short track is a basic thing a player should allow.

## What Changes

- Every playlist row in the popup window and the TUI shows the track's length (`m:ss`, the same format as the playing row), right-aligned, as the playing row does today.
- Lengths are read from the audio files' own headers for MP3, FLAC, Ogg Vorbis/Opus, M4A/MP4 (including ALAC), WAV and AIFF. Other formats show no length until they play.
- The window and the TUI show the total length of the playlist, marked as incomplete while some lengths are unknown.
- Rows on screen are read first, then the rest of the playlist, all in the background, so opening a 500-track playlist stays instant; lengths and the total fill in as they are read.
- When a track plays, the exact length mpv reports is remembered for that file, which fills in formats wimamp can't read and corrects estimated MP3 lengths.
- Lengths are kept in memory only, for as long as the TUI or window runs. No library database, no cache file, no new dependency, no extra processes.
- The playing row keeps using the length mpv reports, which also covers formats wimamp can't read.

Out of scope: tag-based titles for playlist rows (rows keep showing file names, as before; the ListenBrainz change also leaves this deferred), lengths in CLI output, and remembering lengths across restarts.

## Capabilities

### New Capabilities

### Modified Capabilities
- `playlists`: entries gain a displayed length, with rules for unreadable files and the playing entry, and the playlist gains a total length.

## Impact

- New Go file with small header readers per format, standard library only, plus unit tests on tiny generated fixtures.
- `gui.go` and `tui.go`: draw the length on each row and the total (the window's info box, a footer line in the TUI), and redraw when background reads finish.
- The ListenBrainz change (`add-listenbrainz-scrobbling`) is independent: it reads tags inside mpv for the playing file only, so the two changes share no code and can land in either order.
