-- couple-daily initial schema.
-- Auth is email + password. Phone OTP is not in v1: no phone column, no otp_challenges.
-- Apply: psql "$DATABASE_URL" -f migrations/000001_init.up.sql
-- Compatible with golang-migrate / goose filename pairs.

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email citext NOT NULL,
    password_hash varchar(255) NOT NULL,
    display_name varchar(64) NOT NULL,
    avatar_media_id uuid NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_email_len_chk CHECK (char_length(email::text) BETWEEN 3 AND 320),
    CONSTRAINT users_password_hash_chk CHECK (char_length(password_hash) > 0),
    CONSTRAINT users_display_name_chk CHECK (char_length(display_name) BETWEEN 1 AND 64)
);

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id),
    token_hash varchar(64) NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT refresh_tokens_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE couple_spaces (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_count smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT couple_spaces_member_count_chk CHECK (member_count >= 0 AND member_count <= 2)
);

CREATE TABLE space_members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    user_id uuid NOT NULL REFERENCES users (id),
    role varchar(16) NOT NULL DEFAULT 'member',
    joined_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT space_members_space_user_unique UNIQUE (space_id, user_id),
    CONSTRAINT space_members_user_unique UNIQUE (user_id),
    CONSTRAINT space_members_role_chk CHECK (role IN ('owner', 'member'))
);

CREATE INDEX space_members_space_id_idx ON space_members (space_id);

-- Invite creation inserts the space (member_count = 1) and this row together.
CREATE TABLE invite_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code char(6) NOT NULL,
    inviter_user_id uuid NOT NULL REFERENCES users (id),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    status varchar(16) NOT NULL,
    expires_at timestamptz NOT NULL,
    accepted_by_user_id uuid NULL REFERENCES users (id),
    accepted_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT invite_codes_code_unique UNIQUE (code),
    CONSTRAINT invite_codes_code_chk CHECK (code ~ '^[A-Z0-9]{6}$'),
    CONSTRAINT invite_codes_status_chk CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'))
);

CREATE INDEX invite_codes_inviter_status_idx ON invite_codes (inviter_user_id, status);
CREATE INDEX invite_codes_pending_expiry_idx ON invite_codes (expires_at) WHERE status = 'pending';

CREATE TABLE media_assets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_user_id uuid NOT NULL REFERENCES users (id),
    space_id uuid NULL REFERENCES couple_spaces (id),
    content_type varchar(64) NOT NULL,
    byte_size integer NOT NULL,
    storage_key varchar(512) NOT NULL,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_assets_storage_key_unique UNIQUE (storage_key),
    CONSTRAINT media_assets_content_type_chk CHECK (content_type IN ('image/jpeg', 'image/png', 'image/heic', 'image/webp')),
    CONSTRAINT media_assets_byte_size_chk CHECK (byte_size >= 1 AND byte_size <= 10485760),
    CONSTRAINT media_assets_status_chk CHECK (status IN ('pending_upload', 'ready', 'failed'))
);

CREATE INDEX media_assets_uploader_idx ON media_assets (uploader_user_id);
CREATE INDEX media_assets_space_idx ON media_assets (space_id);

ALTER TABLE users
    ADD CONSTRAINT users_avatar_media_id_fkey
    FOREIGN KEY (avatar_media_id) REFERENCES media_assets (id);

CREATE TABLE posts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    author_user_id uuid NOT NULL REFERENCES users (id),
    body text NOT NULL DEFAULT '',
    location_lat double precision NULL,
    location_lng double precision NULL,
    location_name varchar(256) NULL,
    location_place_id varchar(128) NULL,
    location_visibility varchar(16) NOT NULL DEFAULT 'exact',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CONSTRAINT posts_location_pair_chk CHECK ((location_lat IS NULL) = (location_lng IS NULL)),
    CONSTRAINT posts_location_visibility_chk CHECK (location_visibility IN ('exact', 'city', 'hidden'))
);

-- Double-empty (blank body AND zero media) is enforced in the service, not SQL.
CREATE INDEX posts_space_timeline_idx ON posts (space_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE post_media (
    post_id uuid NOT NULL REFERENCES posts (id),
    media_id uuid NOT NULL REFERENCES media_assets (id),
    sort_order smallint NOT NULL,
    PRIMARY KEY (post_id, media_id),
    CONSTRAINT post_media_sort_unique UNIQUE (post_id, sort_order),
    CONSTRAINT post_media_sort_chk CHECK (sort_order >= 0 AND sort_order <= 8)
);

CREATE TABLE likes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    post_id uuid NOT NULL REFERENCES posts (id),
    user_id uuid NOT NULL REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT likes_post_user_unique UNIQUE (post_id, user_id)
);

CREATE INDEX likes_post_id_idx ON likes (post_id);
CREATE INDEX likes_space_post_idx ON likes (space_id, post_id);

CREATE TABLE comments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    post_id uuid NOT NULL REFERENCES posts (id),
    author_user_id uuid NOT NULL REFERENCES users (id),
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CONSTRAINT comments_body_chk CHECK (char_length(body) BETWEEN 1 AND 2000)
);

CREATE INDEX comments_post_timeline_idx ON comments (post_id, created_at ASC, id ASC) WHERE deleted_at IS NULL;

CREATE TABLE restaurant_checkins (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    creator_user_id uuid NOT NULL REFERENCES users (id),
    name varchar(128) NOT NULL,
    address varchar(256) NULL,
    lat double precision NULL,
    lng double precision NULL,
    place_id varchar(128) NULL,
    checked_in_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CONSTRAINT restaurant_checkins_name_chk CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT restaurant_checkins_location_pair_chk CHECK ((lat IS NULL) = (lng IS NULL))
);

CREATE INDEX restaurant_checkins_space_idx ON restaurant_checkins (space_id, checked_in_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE restaurant_favorites (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    creator_user_id uuid NOT NULL REFERENCES users (id),
    name varchar(128) NOT NULL,
    address varchar(256) NULL,
    lat double precision NULL,
    lng double precision NULL,
    place_id varchar(128) NULL,
    note varchar(512) NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CONSTRAINT restaurant_favorites_name_chk CHECK (char_length(name) BETWEEN 1 AND 128)
);

CREATE INDEX restaurant_favorites_space_idx ON restaurant_favorites (space_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX restaurant_favorites_place_unique
    ON restaurant_favorites (space_id, place_id)
    WHERE place_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE anniversaries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id uuid NOT NULL REFERENCES couple_spaces (id),
    creator_user_id uuid NOT NULL REFERENCES users (id),
    title varchar(128) NOT NULL,
    event_date date NOT NULL,
    recurrence varchar(16) NOT NULL DEFAULT 'yearly',
    remind boolean NOT NULL DEFAULT true,
    remind_days_before smallint NOT NULL DEFAULT 0,
    remind_local_time time NOT NULL DEFAULT '09:00',
    timezone varchar(64) NOT NULL DEFAULT 'Asia/Shanghai',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CONSTRAINT anniversaries_title_chk CHECK (char_length(title) BETWEEN 1 AND 128),
    CONSTRAINT anniversaries_recurrence_chk CHECK (recurrence IN ('yearly', 'once')),
    CONSTRAINT anniversaries_remind_days_chk CHECK (remind_days_before >= 0 AND remind_days_before <= 30)
);

CREATE INDEX anniversaries_space_date_idx ON anniversaries (space_id, event_date);
CREATE INDEX anniversaries_remind_idx ON anniversaries (remind) WHERE remind = true AND deleted_at IS NULL;

CREATE TABLE device_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id),
    platform varchar(16) NOT NULL,
    token varchar(512) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT device_tokens_user_token_unique UNIQUE (user_id, token),
    CONSTRAINT device_tokens_platform_chk CHECK (platform IN ('ios', 'android'))
);

CREATE INDEX device_tokens_user_id_idx ON device_tokens (user_id);
