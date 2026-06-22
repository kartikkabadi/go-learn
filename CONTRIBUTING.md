# Contributing to go-learn

Thanks for your interest! go-learn is an open-source Go learning platform built for beginners, by the community.

## Getting Started

1. Fork the repo
2. `make serve` to start the dev server at http://127.0.0.1:4173/
3. `make test` to run tests
4. `make build` to compile

## Adding a Lesson

1. Create `content-bundles/000N-your-lesson.json`
2. Run `make import` to load it into SQLite
3. Open `/lessons/your-lesson-slug` to verify

### Lesson JSON Format

See existing bundles in `content-bundles/` for reference. Each bundle contains:
- `lesson` — id, title, slug, summary, sortOrder
- `sections` — HTML body content with headings
- `questions` — quiz questions with options
- `exercises` — practice module references
- `references` — external links
- `answers` — correct answer keys for auto-seeding

## Adding an Exercise

1. Create `practice/your-exercise/` with `go.mod` and `main.go`
2. Add an exercise entry to the relevant content bundle
3. Run `make import`

## Code Style

- Run `gofmt` before committing
- All exported symbols need doc comments
- Tests use table-driven patterns where practical
- Keep methods on the `Store` interface for D1 compatibility

## Pull Request Process

1. Open an issue describing your change first
2. Create a feature branch
3. Ensure all tests pass (`make test`)
4. Submit a PR with a clear description
