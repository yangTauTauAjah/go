-- =====================================================================
-- SIAKAD Mini - Schema tabel users
-- =====================================================================
-- Tabel users menyimpan akun admin dan mahasiswa.
-- Tabel students berelasi 1-1 ke users melalui kolom user_id.
-- =====================================================================
CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    email       VARCHAR(120) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    role        VARCHAR(20)  NOT NULL CHECK (role IN ('admin', 'mahasiswa')),
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Keunikan email tanpa membedakan huruf besar/kecil.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_key
    ON users (LOWER(email));

CREATE INDEX IF NOT EXISTS users_role_idx
    ON users (role);