## MODIFIED Requirements

### Requirement: Queue view
The TUI and the popup window SHALL switch the playlist list to a view of the queue and back. The view SHALL list the queued entries in play order, in the same row format as the playlist. In the window the panel title SHALL read `WIMAMP QUEUE`; the TUI SHALL show a header line with the number of queued entries. An empty queue SHALL show a one-line hint. In the view, navigation and the mouse SHALL work as in the playlist; playing an entry SHALL play it; a single `d` SHALL remove the selected entry from the queue, never from the playlist, and the queue key SHALL do nothing, since it is easily pressed to go back; and the user SHALL be able to move the selected entry one place later or earlier in the queue. Esc SHALL return to the playlist. Player controls SHALL work as usual in the view.

#### Scenario: Reorder the queue
- **WHEN** entries A and B are queued, in that order, and the user selects B in the queue view and moves it earlier
- **THEN** the queue view lists B before A, and B plays first

#### Scenario: Remove in the queue view
- **WHEN** the user presses `d` on an entry in the queue view
- **THEN** the entry leaves the queue and stays in the playlist

#### Scenario: Queue key in the queue view
- **WHEN** the user presses `e` in the queue view
- **THEN** nothing changes
