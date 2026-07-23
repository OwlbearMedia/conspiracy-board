CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    password_hash text   NOT NULL,
    display_name  text   NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash text PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE boards (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX boards_owner_id_idx ON boards (owner_id);

CREATE TABLE board_members (
    board_id uuid NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    user_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role     text NOT NULL CHECK (role IN ('viewer', 'editor')),
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (board_id, user_id)
);
CREATE INDEX board_members_user_id_idx ON board_members (user_id);

CREATE TABLE invitations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    uuid   NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    email       citext NOT NULL,
    role        text   NOT NULL CHECK (role IN ('viewer', 'editor')),
    token_hash  text   NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    accepted_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invitations_board_id_idx ON invitations (board_id);

CREATE TABLE pins (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id   uuid  NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    kind       text  NOT NULL CHECK (kind IN ('note', 'photo', 'link')),
    content    jsonb NOT NULL DEFAULT '{}',
    x          real  NOT NULL DEFAULT 0,
    y          real  NOT NULL DEFAULT 0,
    z_index    int   NOT NULL DEFAULT 0,
    version    int   NOT NULL DEFAULT 1,
    updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX pins_board_id_idx ON pins (board_id);

CREATE TABLE connections (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    uuid NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    from_pin_id uuid NOT NULL REFERENCES pins (id) ON DELETE CASCADE,
    to_pin_id   uuid NOT NULL REFERENCES pins (id) ON DELETE CASCADE,
    label       text,
    color       text NOT NULL DEFAULT '#c0392b',
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (from_pin_id <> to_pin_id)
);
CREATE INDEX connections_board_id_idx ON connections (board_id);
CREATE UNIQUE INDEX connections_pin_pair_idx
    ON connections (board_id, LEAST(from_pin_id, to_pin_id), GREATEST(from_pin_id, to_pin_id));
