-- Multi-user auth: users, sessions, and per-user scoping for answers and submissions.
-- This is a breaking migration: existing single-user answers and exercise_submissions
-- are dropped because they have no user_id to assign to. Users re-answer quizzes after
-- creating an account. The canonical historical record lives in learning-records/*.md.

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Recreate answers with (user_id, question_id) composite primary key.
-- Nothing references answers, so a plain DROP is safe.
DROP TABLE IF EXISTS answers;
CREATE TABLE answers (
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id  TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    picked_key   TEXT NOT NULL,
    picked_label TEXT NOT NULL DEFAULT '',
    correct      INTEGER NOT NULL,
    answered_at  TEXT NOT NULL,
    PRIMARY KEY (user_id, question_id)
);

-- Recreate exercise_submissions with (user_id, exercise_id) composite primary key.
DROP TABLE IF EXISTS exercise_submissions;
CREATE TABLE exercise_submissions (
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id  TEXT NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    output       TEXT NOT NULL,
    submitted_at TEXT NOT NULL,
    PRIMARY KEY (user_id, exercise_id)
);
