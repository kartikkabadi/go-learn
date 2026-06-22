-- Track whether an exercise submission produced the expected output.
ALTER TABLE exercise_submissions ADD COLUMN correct INTEGER NOT NULL DEFAULT 1;
