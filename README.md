# go-learn — Learn Go from Scratch

**Interactive lessons, quizzes, and exercises** — a server-rendered Go learning platform built for absolute beginners. No prior programming experience required.

[![Go version](https://img.shields.io/github/go-mod/go-version/example/go-learn)]() [![Build](https://img.shields.io/github/actions/workflow/status/example/go-learn/ci.yml)]() [![License](https://img.shields.io/github/license/example/go-learn)]() [![Go Report Card](https://goreportcard.com/badge/github.com/example/go-learn)]()

---

## Quick Start

```bash
make import   # load lessons into SQLite
make serve    # open http://127.0.0.1:4173/
```

That's it. Open your browser and start learning.

## Features

- **Server-rendered lessons** — content delivered as HTML, no JavaScript framework needed
- **Interactive quizzes** — multiple choice and text answers with instant feedback via HTMX
- **Progress tracking** — quiz scores, exercise submissions, and weak spot identification
- **Practice exercises** — real Go programs to write in your terminal
- **Dark mode** — respects your system theme
- **SEO-optimized** — slug URLs, JSON-LD structured data, sitemap, Open Graph tags

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Browser     │────▶│  Go server   │────▶│   SQLite    │
│  (HTMX +     │     │  (net/http)  │     │ (modernc)   │
│   templates) │◀────│  + html/tpl  │◀────│             │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    │  D1 (prod)  │
                    │  (optional) │
                    └─────────────┘
```

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, `net/http`, `html/template` |
| Frontend | HTMX 2.x, vanilla CSS |
| Database | SQLite (`modernc.org/sqlite`) — local dev |
| Database | Cloudflare D1 — production (optional) |
| Deploy | Cloudflare Workers + WASM |
| Auth | None (open platform) |

## Project Structure

```
├── cmd/
│   ├── server/            # Local dev server
│   ├── worker/            # Cloudflare Workers entry point
│   └── import-content/    # JSON bundle import CLI
├── internal/
│   ├── app/               # Config
│   ├── store/             # Data layer (SQLite + D1 interface)
│   │   ├── d1store/       # Workers D1 implementation
│   │   └── migrations/    # SQL schema
│   ├── service/           # Business logic
│   ├── web/
│   │   ├── handlers/      # HTTP handlers
│   │   ├── middleware/     # Logging, recovery
│   │   └── views/         # Templates + renderer
├── content-bundles/       # JSON lesson content
├── practice/              # Exercise modules
├── web/static/            # CSS, images
└── wrangler.toml          # Workers deployment config
```

## Adding a Lesson

1. Create `content-bundles/000N-your-lesson.json` (see existing files for format)
2. Run `make import`
3. Open `/lessons/your-lesson-slug`

## Development

```bash
make test      # run all tests
make build     # compile binaries
make import    # reload lesson content
make serve     # start dev server
```

## Deploy to Cloudflare Workers

```bash
make worker    # build WASM binary
make deploy    # wrangler deploy
```

Requires a Cloudflare account with the Workers Paid plan ($5/mo) and a D1 database.

## License

MIT — see [LICENSE](LICENSE).

## Mission

This project exists to help absolute beginners learn Go. See [MISSION.md](MISSION.md) for the full story.
