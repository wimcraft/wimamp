## Decisions

- **mpv `playlist-move`** does the move. Entry ids stay the same, so the queue, the playing track and track lengths are unaffected, and a saved playlist simply keeps the new order.
- **Same keys, view decides**: `J` / `K` act on the list being shown, so the queue view's behaviour does not change and no new key is used. The existing `queue-later` / `queue-earlier` actions become `move-later` / `move-earlier`; old names in a user's keys file keep working as aliases.
- **One place per press**, no-op at the ends (no wrap-around), matching the queue view.
- Not in scope: moving several marked entries at once, drag and drop in the window, sorting by name or path.
