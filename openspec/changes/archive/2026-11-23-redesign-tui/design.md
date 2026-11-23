## Context

`tui.go` printed `state.line()` (the `wimamp status` format) above the rows, with a footer for the total and extra lines for search and errors. The user picked a boxed "Winamp panels" layout in the window's palette from three mockups (also considered: a flat table like their sonic-tui, and a vim-style status line).

## Goals / Non-Goals

**Goals:** look like the same player as the window; every line exactly the terminal width at any size; no new module.

**Non-Goals:** mouse support, album art, tag-based titles, a marquee for long titles, themes.

## Decisions

**Hand-drawn borders, lipgloss for color.** lipgloss v1 boxes can't put a title or text into a border line, which both boxes need, so borders are built from `╭─╮│╰╯` strings; lipgloss only styles them. Widths come from `ansi.StringWidth` and `ansi.Truncate`, which skip escape codes and count wide characters as two columns. This replaces the rune-counting `rightAlign` and its known CJK misalignment.

**The window's palette as true colors.** Styles are built from the color variables in `gui.go` (`cGreen`, `cWhite`, `cSel`, `cGold`, `cDim`, `cErr`, `cBtn` for borders), so both interfaces match. lipgloss downgrades them on terminals without true color.

**Fixed player height, the playlist takes the rest.** The player box is 5 lines; the playlist box gets everything else, and search, errors and hints live in its bottom border, so the number of rows no longer changes while searching. Below 24 columns the TUI draws at 24.

**Help in the playlist box.** `?` shows the key list where the rows were, titled `KEYS`, instead of replacing the whole screen.

## Risks / Trade-offs

- [Terminals without true color] → lipgloss maps the colors to 256 or 16 colors; the layout doesn't depend on color.
- [Fonts without `❚` `▮` `●`] → all are in common monospace fonts (Menlo, Cascadia, DejaVu); they degrade to a fallback glyph, not to a broken layout.
- [Five lines less for the playlist] → acceptable on normal terminal heights; the user chose the panel layout over the denser status-line one.

## Migration Plan

None: drawing only.
