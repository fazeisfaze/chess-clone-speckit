# Data Model: Chess UI Polish

This feature introduces no persisted entities or schema changes. It reuses existing match
response data and the current chess position.

## Invitation Link

| Attribute | Meaning and validation |
|---|---|
| `invite_url` | Existing complete invitation value returned when a host creates a match. It must be present before the copy action is offered. |
| copy feedback | Transient UI state indicating success or failure of the latest user-requested copy operation; success is set only after clipboard completion. |

## Piece Side (presentation projection)

| Attribute | Meaning and validation |
|---|---|
| piece code | Existing position character identifying a chess piece; uppercase denotes White and lowercase denotes Black. |
| side | Derived `white` or `black` owner used for consistent piece presentation; independent of board orientation and square color. |

## Relationships and lifecycle

- Invitation link belongs to the match creation response and is available in the host's
  waiting-lobby flow.
- Copy feedback belongs only to the current page interaction; it is not persisted.
- Piece side is derived from the existing position on each render and changes only when the
  authoritative position changes (including accepted moves, captures, and promotions).
