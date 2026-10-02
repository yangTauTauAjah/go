-- =====================================================================
-- SIAKAD Mini - Schema tabel enrollments
-- =====================================================================
-- Tabel enrollments menyimpan data mata kuliah yang diambil mahasiswa
-- (Kartu Rencana Studi / KRS).
-- Unique constraint menjamin satu mahasiswa hanya dapat mengambil satu
-- mata kuliah yang sama pada tahun akademik yang sama.
-- =====================================================================
CREATE TABLE IF NOT EXISTS enrollments (
    id              SERIAL PRIMARY KEY,
    student_id      INTEGER     NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    course_id       INTEGER     NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    tahun_akademik  VARCHAR(20) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_unique_per_year
        UNIQUE (student_id, course_id, tahun_akademik)
);

CREATE INDEX IF NOT EXISTS enrollments_student_id_idx ON enrollments(student_id);
CREATE INDEX IF NOT EXISTS enrollments_course_id_idx ON enrollments(course_id);
CREATE INDEX IF NOT EXISTS enrollments_tahun_idx ON enrollments(tahun_akademik);