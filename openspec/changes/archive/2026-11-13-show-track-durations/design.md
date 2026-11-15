## Context

Playlist rows come from mpv's `playlist` property, which carries only `filename`, an optional `title` and the `current` flag (`entry` in `mpv.go`). mpv reports `duration` for the loaded file only and has no command to probe other entries. Both interfaces poll `fetch()` about once a second; the window is a resident process on macOS, the TUI lives until closed. Filenames are absolute paths, because `collect` resolves them before handing them to mpv; `.m3u` entries can be URLs. See proposal.md for motivation and specs/playlists for the rules.

## Goals / Non-Goals

**Goals:**
- Read a length in well under a millisecond of I/O per typical file: headers only, never a full-file scan.
- Standard library only; no cgo.
- Shared by the TUI and the window, with no state outside the running process.

**Non-Goals:**
- Exact lengths for VBR MP3 files without a Xing/Info or VBRI header (estimated, see Risks).
- Formats outside the listed ones (AAC/ADTS, WMA, APE, WavPack, Matroska/WebM).
- Reading tags.

## Decisions

**Parse headers in Go instead of asking mpv.** A second mpv per file (`--frames=0`, print `duration`) costs roughly 50–100 ms per process; 500 files is most of a minute and hundreds of spawns. Loading each file into the running mpv would disturb playback. ffprobe or a tag library would add a dependency. Header parsing is a few dozen lines per format and runs in microseconds per file. Per format:
- MP3: skip an ID3v2 tag (size from its header), find the first frame sync, decode version/layer/sample rate. Use the Xing/Info frame count or the VBRI frame count when present; otherwise assume CBR and estimate from the audio byte count and the first frame's bitrate (minus a trailing 128-byte ID3v1 tag).
- FLAC: `fLaC` marker, STREAMINFO block: total samples / sample rate.
- Ogg Vorbis and Opus: identification header for the sample rate (Opus: always 48 kHz, minus pre-skip); granule position of the last page, found by reading the final 64 KiB and searching backwards for `OggS`.
- M4A/MP4: walk top-level boxes to `moov`, then `mvhd`: duration / timescale (versions 0 and 1). If `moov` is at the end of the file, seek to it by box sizes; no full read.
- WAV: RIFF chunks: `data` size / byte rate from `fmt `.
- AIFF: `COMM` chunk: frame count / 80-bit extended sample rate.
A file is identified by its extension first, falling back to nothing (no content sniffing): extensions are how `add` selects files already.

**One function, one cache.** `trackLength(path) (seconds float64, ok bool)` dispatches by extension. A process-wide cache maps path to `{size, mtime, seconds, ok}`; a `Stat` that matches the stored size and mtime reuses the entry, so an edited file is re-read and a deleted one turns into "no length". Failed reads are cached too, so an unreadable file is tried once per change rather than on every frame.

**Visible rows first, then the rest, on one background worker.** The total needs every length, so the whole playlist is read, but rows on screen jump the queue: when a row is drawn without a cache entry its path goes to the front of a deduplicated queue, and on each playlist change all unknown paths go to the back. A single background goroutine drains the queue and wakes the interface after each batch (at most a few times a second): the window sets `dirty` and calls `ebiten.ScheduleFrame`, the TUI receives a message that triggers a render. One worker keeps disk access sequential, which suits spinning disks and network shares. At roughly 0.1 ms per file on an SSD, a 572-track playlist is read in well under a second, once per process; the window stays resident, so in practice once per login.

**Learning from playback.** Whenever `fetch()` reports a positive `duration` for the playing entry, it is written to the cache for that path, marked as exact, with the file's current size and mtime. A later header read never overwrites an exact entry for an unchanged file. This fills formats the readers skip and corrects CBR estimates for headerless VBR MP3s.

**No disk cache.** Considered: persisting lengths keyed by a content hash or by path+size+mtime. Rejected: a content hash reads the whole file, far more than the header read it would save; a path+size+mtime file saves under a second per process start on typical libraries, but adds a file that can go stale, needs locking between the TUI and the window, and is the start of a library database the project avoids. The only loss is that lengths learned from playback for unreadable formats are forgotten on restart.

**Total.** Computed on each draw from the playlist and the cache (a map lookup per entry; 10,000 entries is negligible next to drawing), so it is never out of sync with adds and removes. Window: appended to the info box (`17/572  39:12:05+`). TUI: a footer line below the playlist (`572 tracks  39:12:05+`), since its status line is shared with `wimamp status` and stays unchanged. Totals use `h:mm:ss` from one hour, because `fmtTime`'s `m:ss` turns a 40-hour playlist into `2400:00`.

**Rows show the cache, the playing row shows mpv.** Row rendering asks the cache only; the playing row keeps using `state.Dur` as today, so formats wimamp can't parse still get a length while they play. Search results reuse the same row drawing and so get lengths too.

## Risks / Trade-offs

- [VBR MP3 without a Xing/VBRI header gets a CBR estimate that can be off] → most encoders (LAME, iTunes, FFmpeg) write the Xing/Info header; the playing row shows mpv's exact value.
- [A malformed file could make a parser read garbage sizes] → every parser bounds its reads (no allocation from untrusted sizes beyond a small fixed buffer), treats any inconsistency as "no length", and is fuzzed briefly with `go test -fuzz` on the fixtures as part of the tests.
- [Files on a slow network share block the worker] → the worker is off the UI goroutine, so the interface never waits; lengths simply appear later.
- [Memory] → one small struct per distinct path seen; a 10,000-track playlist is well under a megabyte.
- [Reading every file on a slow network share] → takes longer but runs in the background; the total stays marked `+` until done.

## Migration Plan

None: display-only, nothing stored. Rollback is reverting the change.
