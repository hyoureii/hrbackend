package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email string `gorm:"not null,unique"`
	Password string `gorm:"not null"`
}

type RefreshToken struct {
	gorm.Model
	UserId uint `gorm:"foreignkey"`
}
-- 1. users table -------------------------------------------------
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- 2. refresh_tokens table ----------------------------------------
CREATE TABLE refresh_tokens (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id        TEXT         NOT NULL,          -- UUID or random string
    token_hash       TEXT         NOT NULL,          -- SHA-256 of raw JWT
    issued_at        TIMESTAMPTZ  DEFAULT NOW(),
    expires_at       TIMESTAMPTZ  NOT NULL,
    first_issued_at  TIMESTAMPTZ  DEFAULT NOW(),     -- for global cap
    UNIQUE (user_id, device_id, token_hash)
);

-- help quickly find rows for revocation
CREATE INDEX idx_refresh_user_device ON refresh_tokens(user_id, device_id);
CREATE INDEX idx_refresh_token_hash  ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_expires     ON refresh_tokens(expires_at);

-- 3. optional clean-up job ---------------------------------------
-- DELETE FROM refresh_tokens WHERE expires_at < NOW();
