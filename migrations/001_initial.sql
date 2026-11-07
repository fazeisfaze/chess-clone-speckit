CREATE TABLE matches (
    id text PRIMARY KEY,
    invite_token_digest bytea NOT NULL UNIQUE,
    status text NOT NULL CHECK (status IN ('waiting', 'active', 'completed', 'unavailable')),
    white_player_id text,
    black_player_id text,
    position text NOT NULL DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    time_control text NOT NULL CHECK (time_control IN ('3+2', '5+0', '10+0')),
    initial_time_ms bigint NOT NULL CHECK (initial_time_ms > 0),
    increment_ms bigint NOT NULL CHECK (increment_ms >= 0),
    white_remaining_ms bigint NOT NULL CHECK (white_remaining_ms >= 0),
    black_remaining_ms bigint NOT NULL CHECK (black_remaining_ms >= 0),
    clock_active_color text CHECK (clock_active_color IN ('white', 'black')),
    clock_started_at timestamptz,
    version bigint NOT NULL DEFAULT 0 CHECK (version >= 0),
    result jsonb,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (
        (status = 'active' AND clock_active_color IS NOT NULL AND clock_started_at IS NOT NULL)
        OR (status <> 'active' AND clock_active_color IS NULL AND clock_started_at IS NULL)
    ),
    CHECK (white_player_id IS NULL OR white_player_id <> black_player_id)
);

CREATE TABLE player_seats (
    id text PRIMARY KEY,
    match_id text NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    color text NOT NULL CHECK (color IN ('white', 'black')),
    recovery_credential_digest bytea NOT NULL UNIQUE,
    connected boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (match_id, color),
    UNIQUE (match_id, id)
);

CREATE TABLE moves (
    match_id text NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    sequence bigint NOT NULL CHECK (sequence > 0),
    player_seat_id text NOT NULL,
    from_square text NOT NULL CHECK (from_square ~ '^[a-h][1-8]$'),
    to_square text NOT NULL CHECK (to_square ~ '^[a-h][1-8]$'),
    promotion text CHECK (promotion IN ('q', 'r', 'b', 'n')),
    notation text NOT NULL,
    position_after text NOT NULL,
    accepted_at timestamptz NOT NULL,
    PRIMARY KEY (match_id, sequence),
    FOREIGN KEY (match_id, player_seat_id) REFERENCES player_seats(match_id, id)
);

CREATE TABLE draw_offers (
    id text PRIMARY KEY,
    match_id text NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    offering_seat_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'accepted', 'declined')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    resolved_at timestamptz,
    FOREIGN KEY (match_id, offering_seat_id) REFERENCES player_seats(match_id, id),
    CHECK ((status = 'pending') = (resolved_at IS NULL))
);

CREATE UNIQUE INDEX one_pending_draw_offer_per_match
    ON draw_offers(match_id) WHERE status = 'pending';

CREATE INDEX moves_match_accepted_at ON moves(match_id, accepted_at);
