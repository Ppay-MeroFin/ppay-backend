CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS user_pins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_e164 TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    pin_hash TEXT NOT NULL,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_pins_phone_e164_not_blank CHECK (length(trim(phone_e164)) > 0),
    CONSTRAINT user_pins_full_name_not_blank CHECK (length(trim(full_name)) >= 2),
    CONSTRAINT user_pins_failed_attempts_nonnegative CHECK (failed_attempts >= 0)
);

CREATE INDEX IF NOT EXISTS idx_user_pins_locked_until
    ON user_pins (locked_until);