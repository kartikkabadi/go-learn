PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS lessons (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    slug       TEXT NOT NULL DEFAULT '',
    summary    TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS lesson_sections (
    id         TEXT PRIMARY KEY,
    lesson_id  TEXT NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    heading    TEXT NOT NULL,
    body_html  TEXT NOT NULL,
    sort_order INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS questions (
    id            TEXT PRIMARY KEY,
    lesson_id     TEXT NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    prompt        TEXT NOT NULL,
    correct_key   TEXT NOT NULL,
    question_type TEXT NOT NULL CHECK (question_type IN ('choice', 'text')),
    section_tag   TEXT NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS question_options (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    option_key  TEXT NOT NULL,
    label       TEXT NOT NULL,
    is_correct  INTEGER NOT NULL DEFAULT 0,
    sort_order  INTEGER NOT NULL,
    UNIQUE (question_id, option_key)
);

CREATE TABLE IF NOT EXISTS answers (
    question_id  TEXT PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
    picked_key   TEXT NOT NULL,
    picked_label TEXT NOT NULL DEFAULT '',
    correct      INTEGER NOT NULL,
    answered_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS glossary_terms (
    id         TEXT PRIMARY KEY,
    term       TEXT NOT NULL,
    definition TEXT NOT NULL,
    sort_order INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS references_ext (
    id        TEXT PRIMARY KEY,
    title     TEXT NOT NULL,
    url       TEXT NOT NULL,
    notes     TEXT NOT NULL DEFAULT '',
    lesson_id TEXT REFERENCES lessons(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS exercises (
    id           TEXT PRIMARY KEY,
    lesson_id    TEXT NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    path         TEXT NOT NULL,
    instructions TEXT NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS exercise_submissions (
    exercise_id  TEXT PRIMARY KEY REFERENCES exercises(id) ON DELETE CASCADE,
    output       TEXT NOT NULL,
    submitted_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS insights (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('weak_spot', 'preference', 'milestone')),
    active     INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS mission (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    why              TEXT NOT NULL,
    success_criteria TEXT NOT NULL,
    constraints_text TEXT NOT NULL
);
