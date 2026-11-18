## Decisions

- **fzf's own walker** (`--walker=file|dir,follow --walker-root=$HOME`) instead of feeding it a list: no file walk in wimamp, results stream in while typing, and the same flags work on Windows. Needs fzf 0.48+.
- **Clear `FZF_DEFAULT_COMMAND`**: when set, fzf uses it instead of the walker and ignores `--walker-root`.
- **Skip `Library` and `AppData`** with `--walker-skip`, besides fzf's default `.git,node_modules`: on macOS and Windows they hold huge numbers of non-music files.
- **`tea.ExecProcess`** suspends the TUI and gives fzf the terminal; the chosen paths come back as a message and are loaded through the player queue, off the UI loop, since a big folder can take a while to walk.
- Root is the home folder. A configurable root or starting from the playing track's folder is left out until home proves annoying.
