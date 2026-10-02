-- =====================================================================
-- SIAKAD Mini - Schema tabel courses
-- =====================================================================
-- Tabel courses menyimpan daftar mata kuliah yang ditawarkan.
-- =====================================================================
CREATE TABLE IF NOT EXISTS courses (
    id          SERIAL PRIMARY KEY,
    kode_mk     VARCHAR(20)  NOT NULL UNIQUE,
    nama_mk     VARCHAR(150) NOT NULL,
    sks         INTEGER      NOT NULL CHECK (sks BETWEEN 1 AND 6),
    semester    INTEGER      NOT NULL CHECK (semester BETWEEN 1 AND 14),
    kuota       INTEGER      NOT NULL CHECK (kuota > 0),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS courses_semester_idx ON courses(semester);
CREATE INDEX IF NOT EXISTS courses_kode_mk_idx ON courses(kode_mk);