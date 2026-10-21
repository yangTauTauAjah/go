CREATE TABLE IF NOT EXISTS users (
 id SERIAL PRIMARY KEY,
 username VARCHAR(50) NOT NULL,
 email VARCHAR(255) NOT NULL,
 password VARCHAR(255) NOT NULL,
 is_active BOOLEAN NOT NULL DEFAULT TRUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Keunikan username tanpa membedakan huruf besar dan kecil.
-- Inilah yang menggantikan pemeriksaan manual di pertemuan 2.
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key
 ON users (LOWER(username));
CREATE INDEX IF NOT EXISTS users_email_lower_idx
 ON users (LOWER(email));

-- Urutan column pada index HARUS sama persis dengan ORDER BY pada
-- query, termasuk arah DESC-nya. Bila tidak, PostgreSQL tetap dapat
-- memakai index untuk sebagian kasus, tetapi tidak dapat memakainya
-- untuk melompat langsung ke posisi cursor — dan seluruh keuntungan
-- keyset pagination hilang tanpa satu pun pesan kesalahan.
CREATE INDEX IF NOT EXISTS users_created_at_id_desc_idx
 ON users (created_at DESC, id DESC);
