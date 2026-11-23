## Decisions

- **Tags from the file with `dhowden/tag`**, not mpv: mpv only exposes `metadata` for the playing track, and asking it to open another file would disturb playback. The library is small, pure Go and covers MP3 (ID3v1/v2), M4A, FLAC and OGG/Opus. Hand-writing four tag parsers is not worth it, unlike the length readers, which only needed a few header fields.
- **Format line** comes from what the file gives cheaply: the extension, plus sample rate/bit depth where the length readers already parse headers. No bitrate scan.
- **Read on open, off the UI loop**, through the player-call queue like other file work; the view shows the path and "reading…" until tags arrive. Nothing is cached: one file per press.
- **A view in the playlist panel**, not an overlay: same pattern as the queue view; esc or `i` goes back. Player controls keep working; list navigation is not available while it's shown.
- **Show in folder**: `open -R` on macOS, `explorer /select,` on Windows, `xdg-open` on the parent folder elsewhere. Errors (e.g. over SSH with no desktop) show in the bottom border.
- **Missing tags** show as empty rows left out, not "unknown"; a file without tags shows only path, length and format. URLs show only the URL.

## Not in scope

- Copy path to clipboard (the path is on screen and selectable), web lookup (MusicBrainz / search), always showing artist/album for the playing track in the player panel, using tags for playlist names, cover art, editing tags.
