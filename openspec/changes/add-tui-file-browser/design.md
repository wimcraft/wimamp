## Context

The queue view already swaps the playlist panel's contents and keeps navigation, the scroll window and row drawing. The browser is a third thing that panel can show, with its own key handling while it is open, like search.

## Decisions

- **In the playlist panel, not a popup**: same place, same row style and scrolling as the queue view; the panel title shows the folder (`~`-shortened, cut from the left when too long), the bottom border shows the keys and the number of marked items.
- **Keys while browsing** (local to the browser, everything else is ignored except `?` and `q`):
  - `j` `k` `gg` `G` `ctrl+d` `ctrl+u` and arrows/pg keys move;
  - `l` / `right` / `enter` on a folder open it; `h` / `left` / `backspace` go to the parent, with the folder just left selected; `~` goes home;
  - `space` / `tab` mark or unmark the selected item and move down;
  - `enter` on a file, or `a`, add the marked items, or the selected item when nothing is marked, and close the browser;
  - `esc` closes without adding.
  Marks survive moving between folders and are added in the order marked.
- **What is listed**: folders (shown with a trailing `/`) first, then files with an audio extension (`audioExt`, the same list `wimamp add` uses), each group sorted case-insensitively. Hidden entries are skipped. Symlinks are followed. A folder that can't be read shows its error in the bottom border and the browser stays where it was.
- **Adding** goes through the existing `load(paths, "append")`, off the UI loop, as the fzf picker does now.
- **Start folder** is stored in the player as `user-data/wimamp/browse-dir`, so closing and reopening the TUI (or a second TUI) starts where you left off, without a state file; it is lost when the player quits. Without it: the playing track's folder, then home.
- **fzf removed** rather than kept as a second way: one picker to maintain, and the user found it unhandy. `wimamp add <path>` stays for shell users.

## Not in scope

- Type-to-filter inside a folder (could reuse search later), showing track lengths or tags in the browser, the browser in the popup window, network locations.
