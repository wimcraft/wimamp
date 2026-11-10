## Why

There is no music player that works the same on macOS, Linux and Windows, puts the keyboard first, plays local files offline, and can be controlled from any terminal or hotkey tool. Streaming TUIs like ncspot break behind corporate VPNs and have authorization problems. Wimamp is a small player in the spirit of Winamp, built for exactly that workflow.

## What Changes

- A single `wimamp` binary that starts a background player daemon on first use. The daemon keeps playing after the terminal that started it closes.
- CLI commands to control playback: play, pause, toggle, stop, next, prev, seek, volume, status, add, clear, quit. Any hotkey tool can bind them.
- Playlists built from files and folders, saved and loaded as `.m3u`.
- A TUI with vim-style navigation, opened by running `wimamp` with no arguments.
- Configurable keybindings, shared by the TUI and the popup window, with a built-in key help (`?`, `wimamp keys`).
- Fuzzy jump-to-track search (`/` or `ctrl+j`).
- `wimamp gui`: a Winamp-style popup window with the main panel and the playlist (no equalizer, no visualizer). It opens Spotlight-style and closes on Esc or when it loses focus.
- Out of scope: a media library with library-wide search, tag reading for playlist rows, an equalizer, visualization, lyrics, album art, built-in global hotkeys, streaming services.

## Capabilities

### New Capabilities
- `player-daemon`: the background player's lifecycle: start on demand, a single instance, surviving the terminal, quitting.
- `cli-control`: one-shot commands that control playback and report status, for scripts and hotkey tools.
- `playlists`: adding files and folders to the play queue, and saving, loading and listing named playlists.
- `keybindings`: default keys, user overrides from a config file, and vim-style key sequences.
- `tui`: the terminal interface showing the playlist and playback status.
- `track-search`: fuzzy jump-to-track over the playlist, in the TUI and the window (Winamp's "jump to file").
- `popup-window`: the Winamp-style window with transport controls, a seek bar, volume, shuffle/repeat, and an editable playlist.

### Modified Capabilities

## Impact

- A new Go module in this repository. Dependencies: Bubbletea, Ebitengine, zenity.
- Runtime dependency: `mpv` must be installed and on `PATH`.
- Files on disk: a socket in the user's runtime or cache directory, and `keys.conf` plus `playlists/*.m3u` in the config directory (`$XDG_CONFIG_HOME/wimamp`, `~/.config/wimamp`, or `%AppData%\wimamp` on Windows).
