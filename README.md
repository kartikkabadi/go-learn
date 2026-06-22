# go-learn

A local Go learning platform: lessons, quizzes, practice tracking, and a progress dashboard.

## Quick start

```bash
make import   # load lessons into SQLite (first time or after content changes)
make serve    # start at http://127.0.0.1:4173/
```

## Architecture

- **SQLite** (`progress/go-learn.db`) — lessons, quizzes, answers, exercises, insights
- **Content bundles** (`content-bundles/*.json`) — version-controlled authoring format; imported into DB
- **Go server** (`cmd/server`) — server-rendered pages + HTMX quizzes
- **Practice code** (`practice/`) — standalone Go modules you run in the terminal

## Pages

| URL | Purpose |
|-----|---------|
| `/` | Dashboard — mission, stats, lesson progress, weak spots |
| `/lessons` | Lesson index |
| `/lessons/0003` | Lesson content + quizzes |
| `/progress` | Full quiz history |
| `/reference` | Glossary + external links |
| `/practice` | Submit exercise terminal output |

## Adding a new lesson

1. Create `content-bundles/0004-your-lesson.json`
2. Run `make import`
3. Open `/lessons/0004`

## Development

```bash
make test
make build
```
