## ADDED Requirements

### Requirement: Queue from search
In the search prompt, `ctrl+e` SHALL queue the highlighted match, keep the prompt open with the same query, and highlight the next match, so that several matches can be queued in a row.

#### Scenario: Queue two matches
- **WHEN** the user searches `bad religion`, presses `ctrl+e` twice and then Esc
- **THEN** the first two matches are queued in that order and the playlist is shown again

### Requirement: Why a result matched
When the query matches a folder of an entry rather than its name, the search results SHALL show that folder before the name, as `Folder / Name`, so it is clear why the entry matched.

#### Scenario: Match in the artist folder
- **WHEN** the user searches `religion` and the playlist holds `Bad Religion/1988 - Suffer/03. You Are.mp3`
- **THEN** the result reads `Bad Religion / 03. You Are`
