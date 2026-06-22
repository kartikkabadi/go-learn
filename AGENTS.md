## Learned User Preferences

- Complete beginner — Go is their first programming language; explain jargon before using it
- Solo learning with the agent as teacher; intense pace (~6+ hours/week); no communities for now
- Wants interactive teaching: drills, paste-output checkpoints, and weak-spot tracking across lessons
- Prefers dark mode via system theme on HTML pages; dark background `#181818`
- Prefers production-grade long-term architecture over browser localStorage or other short-term hacks
- Chose Go-native server-rendered pages (templates + HTMX) over a React/Next.js frontend
- Content source of truth is SQLite; JSON import bundles are the version-controlled authoring format
- IDE tab completion is fine; still practice hand-typing `:=` and basic syntax occasionally

## Learned Workspace Facts

- `go-learn` is a personal Go teaching workspace; ground teaching in `MISSION.md`
- Start the app with `make serve` → http://127.0.0.1:4173/ (binds `127.0.0.1:4173`)
- Quiz answers and lesson catalog live in SQLite at `progress/go-learn.db`
- Add or update lessons via `content-bundles/*.json` then `make import`
- `practice/` holds independent Go modules per exercise (not wired to auto-grading in v1)
- Agent session notes go in `learning-records/*.md`; use `command go` if the `sfw` alias interferes
- Lessons must be opened through the server (not `file://`) for quizzes to persist
