## ADDED Requirements

### Requirement: Play queue
The user SHALL be able to queue playlist entries to play next. Queueing an entry SHALL add it to the end of the queue; queueing an entry that is already queued SHALL remove it from the queue. When a track ends on its own, or on `next`, the first queued entry still in the playlist SHALL play and leave the queue; with the queue empty, playback SHALL continue as before. After queued entries have played, the playlist SHALL continue after the last one played, as in Winamp. A queued entry that starts playing for any other reason SHALL leave the queue, and so SHALL an entry removed from the playlist. Queueing SHALL NOT change the playlist order. The queue SHALL work with no wimamp interface open, SHALL last as long as the player runs, and SHALL NOT be saved with the playlist. The TUI and the popup window SHALL show each queued entry's place in the queue before its name, as `[1]`, `[2]` and so on.

#### Scenario: Queued tracks play first
- **WHEN** entry 1 of a four-entry playlist is playing and the user queues entry 4 and then entry 2
- **THEN** the entries play in the order 1, 4, 2, 3

#### Scenario: Queue at the end of the playlist
- **WHEN** the last entry is playing, repeat is off, and entry 1 is queued
- **THEN** entry 1 plays after the last entry ends

#### Scenario: Next takes the queue
- **WHEN** entry 3 is queued and the user presses the next key or runs `wimamp next`
- **THEN** entry 3 plays and leaves the queue

#### Scenario: Unqueue
- **WHEN** the user queues an entry that is already queued
- **THEN** it leaves the queue and stays in the playlist

#### Scenario: Marks
- **WHEN** two entries are queued
- **THEN** the first one's row shows `[1]` and the second one's `[2]` before the name, and each mark disappears when that entry starts playing

### Requirement: Queue view
The TUI and the popup window SHALL switch the playlist list to a view of the queue and back. The view SHALL list the queued entries in play order, in the same row format as the playlist. In the window the panel title SHALL read `WIMAMP QUEUE`; the TUI SHALL show a header line with the number of queued entries. An empty queue SHALL show a one-line hint. In the view, navigation and the mouse SHALL work as in the playlist; playing an entry SHALL play it; removing or queueing an entry SHALL only remove it from the queue, never from the playlist; and the user SHALL be able to move the selected entry one place later or earlier in the queue. Esc SHALL return to the playlist. Player controls SHALL work as usual in the view.

#### Scenario: Reorder the queue
- **WHEN** entries A and B are queued, in that order, and the user selects B in the queue view and moves it earlier
- **THEN** the queue view lists B before A, and B plays first

#### Scenario: Remove in the queue view
- **WHEN** the user presses the remove key on an entry in the queue view
- **THEN** the entry leaves the queue and stays in the playlist
