## 1. Player

- [x] 1.1 `queue.lua`: pop the first queued id after a natural end (before-start hook, idle at the end of the playlist), drop started and removed entries, set a marker property
- [x] 1.2 Embed the script, write it to the cache dir and pass `--script` when starting mpv; load it into an older player on the first queue change
- [x] 1.3 Entry ids and the queue in `fetch()`; `toggleID`, `unqueueID`, `moveID`, `queueIndices`; `next` plays the queue head
- [x] 1.4 Integration test against a private mpv: queue order, queue after the last track, `next`, a queued entry picked by hand

## 2. Interfaces

- [x] 2.1 Actions and default keys: `queue` (`e`), `queue-view` (`E`), `queue-later` / `queue-earlier` (`J` / `K`); help rows
- [x] 2.2 Search: `ctrl+e` queues the highlighted match, keeps the query and moves to the next match; results matched by folder show that folder
- [x] 2.3 `[n]` marks on queued rows in the window and the TUI
- [x] 2.4 Queue view in the window (title, rows, selection, scrollbar, mouse, info box, Esc back) and the TUI (header line, rows, Esc back); `e`/`dd`/REM unqueue there
- [x] 2.5 Check by hand on macOS and Windows: queue from search, marks clear as tracks play, works with the window closed and with shuffle on, queue view reorder and unqueue

## 3. Docs

- [x] 3.1 README: the queue and its keys
