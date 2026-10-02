-- =====================================================================
-- SIAKAD Mini - Schema tabel students
-- =====================================================================
-- Tabel students menyimpan data akademik mahasiswa.
-- Relasi 1-1 ke users (via user_id) dan 1-N ke enrollments.
-- =====================================================================
CREATE TABLE IF NOT EXISTS students (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER      NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    nim          VARCHAR(12)  NOT NULL UNIQUE,
    nama         VARCHAR(120) NOT NULL,
    prodi        VARCHAR(100) NOT NULL,
    angkatan     INTEGER      NOT NULL CHECK (angkatan BETWEEN 1900 AND EXTRACT(YEAR FROM NOW())::INT),
    ipk_terakhir NUMERIC(3,2) NOT NULL DEFAULT 0.00 CHECK (ipk_terakhir BETWEEN 0 AND 4),
    deleted_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS students_nim_idx ON students(nim);
CREATE INDEX IF NOT EXISTS students_prodi_idx ON students(prodi);
CREATE INDEX IF NOT EXISTS students_angkatan_idx ON students(angkatan);
CREATE INDEX IF NOT EXISTS students_deleted_at_idx ON students(deleted_at);