CREATE TABLE IF NOT EXISTS achievements (
    id SERIAL PRIMARY KEY,
    student_id INT NOT NULL,
    achievement_name VARCHAR(255) NOT NULL,
    achievement_score NUMERIC(5, 2)   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_achievement_student
        FOREIGN KEY (student_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS achievements_student_id_idx
    ON achievements (student_id);

CREATE INDEX IF NOT EXISTS achievements_created_at_idx
    ON achievements (created_at DESC);
