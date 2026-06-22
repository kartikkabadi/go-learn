# Teaching Notes

## Learner profile (session 0)

- Complete beginner — no prior programming experience
- Go is their first language
- Chose Go partly because people say it's approachable
- Workspace is greenfield: no git, no existing files

## Preferences

- Solo learner — no communities for now
- Intense pace (~6+ hours/week)
- Success metric: write and run small programs from scratch
- Wants **more interactive** teaching: drills, paste-output checkpoints, weak-spot tracking
- IDE tab completion is fine; still practice hand-typing `:=` and basic syntax
- Prefers **dark mode** — system theme on HTML pages; dark bg `#181818`
- Quiz answers in **SQLite** (`progress/go-learn.db`), not browser localStorage

## Teaching tracker

| What | Where |
|------|-------|
| Everything | `make serve` → http://127.0.0.1:4173/ |
| Quiz answers + content | `progress/go-learn.db` |
| Content authoring | `content-bundles/*.json` → `make import` |
| Practice code | `practice/` |
| Agent notes | `learning-records/*.md` |

## Mission drivers (from intake)

- Build a side project eventually
- Automate boring tasks
- Understand how software actually works
- Go chosen because recommended as beginner-friendly

## Environment

- Go 1.26.2 installed at `/usr/local/go/bin/go` (use `command go` if `sfw` alias interferes)
