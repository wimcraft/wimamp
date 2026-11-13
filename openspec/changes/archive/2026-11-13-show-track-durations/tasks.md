## 1. Length readers

- [x] 1.1 Add `trackLength(path)` dispatching by extension, returning no length for unknown extensions, URLs and unreadable files
- [x] 1.2 FLAC (STREAMINFO) and WAV/AIFF (chunk walk) readers
- [x] 1.3 Ogg Vorbis and Opus reader (last-page granule, Opus pre-skip)
- [x] 1.4 M4A/MP4 reader (`moov`/`mvhd`, versions 0 and 1, `moov` at either end)
- [x] 1.5 MP3 reader (ID3v2 skip, Xing/Info and VBRI frame counts, CBR estimate with ID3v1 excluded)
- [x] 1.6 Tests per format on tiny fixtures generated in the test (no binary files in the repo), plus truncated and garbage inputs returning no length; a fuzz target seeded with the fixtures

## 2. Cache and background reading

- [x] 2.1 Process-wide cache keyed by path, invalidated by size and mtime, caching failures too
- [x] 2.2 Single background worker with a deduplicated queue (visible rows to the front, the rest of the playlist to the back) and a wake-up callback; test that a path queued twice is read once and that a front-queued path is read first
- [x] 2.3 Record mpv's duration for the playing entry as exact; test that a header read does not overwrite it for an unchanged file
- [x] 2.4 Total: sum of known lengths with a `+` when any is unknown, `h:mm:ss` from one hour; tests for the spec scenarios

## 3. Interfaces

- [x] 3.1 Popup window: draw the cached length right-aligned on every row (playing row keeps mpv's duration), queue visible rows, show the total in the info box, redraw on wake-up
- [x] 3.2 TUI: same for its rows, plus a footer line with the track count and total, waking the Bubbletea loop with a message
- [x] 3.3 Check by hand: a 500-entry playlist opens instantly in both, lengths fill in, a `.wma` row shows no length until played and then keeps it, the total loses its `+` once all lengths are known, the playing row matches mpv; idle CPU of the window stays within the popup-window spec

## 4. Docs

- [x] 4.1 README: one sentence on playlist lengths and which formats show them
