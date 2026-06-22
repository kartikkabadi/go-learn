# go-learn

[![CI](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml/badge.svg)](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev)
[![Zero JS framework](https://img.shields.io/badge/JS_framework-zero-00ADD8.svg)](#)

A free, interactive Go programming course with lessons, quizzes, and hands-on exercises.
Server-rendered with Go + HTMX. No JavaScript framework, no build step, no dependencies
on the frontend. Deployed on Cloudflare Workers with D1.

**Live**: [go-learn.kartikkabadi.com](https://go-learn.kartikkabadi.com)

```text
Browser (HTMX + CSS)
    │
    │  HTTP requests
    ▼
Go server  ←── net/http, html/template, no JS framework
    │
    │  SQL queries
    ▼
SQLite (dev)  /  Cloudflare D1 (prod)
```

## Why

Most Go tutorials are walls of text. This one is interactive — every lesson has quizzes
with instant feedback, practice exercises you run in your terminal, and progress tracking
that identifies your weak spots. It's built for absolute beginners: no prior programming
experience required.

## Quick start

```bash
# Clone and run locally
git clone https://github.com/kartikkabadi/go-learn.git
cd go-learn
make import   # load lessons into SQLite
make serve    # open http://127.0.0.1:4173/
```

That's it. Open your browser and start learning.

## Features

- **10 interactive lessons** — from "What is a program?" to methods on structs
- **Instant quiz feedback** — multiple choice and text answers via HTMX, no page reloads
- **Progress tracking** — quiz scores, exercise submissions, weak spot identification
- **User accounts** — signup, login, session cookies (bcrypt + secure sessions)
- **Practice exercises** — real Go programs to write in your terminal
- **Dark mode** — respects your system theme
- **Fast** — gzip compression, immutable static asset caching, server-rendered HTML
- **SEO-optimized** — slug URLs, JSON-LD structured data, sitemap with lastmod, OG tags
- **Secure** — CSRF protection, HSTS, strict CSP, rate limiting, bcrypt passwords

## Lessons

| # | Lesson | Topics |
|---|--------|--------|
| 1 | What Is a Program? | Programs, source code, running code, output |
| 2 | Variables & Printing | `:=`, types, `fmt.Println`, string formatting |
| 3 | Decisions with if | Conditions, branching, else, comparison operators |
| 4 | Loops | `for`, range, break, continue |
| 5 | Functions | `func`, parameters, return values, multiple returns |
| 6 | Slices | Arrays vs slices, append, length vs capacity |
| 7 | Maps | Key-value pairs, make, iteration, zero values |
| 8 | Structs | Custom types, fields, embedding |
| 9 | Pointers | `&`, `*`, pointer semantics, when to use them |
| 10 | Methods | Method receivers, value vs pointer receivers |

## Architecture

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, `net/http`, `html/template` |
| Frontend | HTMX 2.x, vanilla CSS, no build step |
| Database | SQLite (`modernc.org/sqlite`) — local dev |
| Database | Cloudflare D1 — production |
| Deploy | Cloudflare Workers (WASM) |
| Auth | bcrypt + session cookies |
| Compression | gzip middleware (dev), Cloudflare auto-gzip (prod) |

## Project structure

```
├── cmd/
│   ├── server/            # Local dev server
│   ├── worker/            # Cloudflare Workers entry point (WASM)
│   └── import-content/    # JSON bundle import CLI
├── internal/
│   ├── app/               # Config, base URL, env vars
│   ├── store/             # Data layer interface + SQLite impl
│   │   ├── d1store/       # Cloudflare D1 implementation
│   │   ├── migrations/    # SQL schema (embedded, auto-applied)
│   │   └── auth.go        # User + session store methods
│   ├── service/           # Business logic (progress, dashboard)
│   ├── web/
│   │   ├── handlers/      # HTTP handlers (lessons, quiz, auth)
│   │   ├── middleware/     # Auth, CSRF, gzip, rate limit, request ID
│   │   └── views/         # Templates + renderer
├── content-bundles/       # JSON lesson content (source of truth)
├── practice/              # Exercise modules (independent Go programs)
├── web/static/            # CSS, HTMX, OG image
└── wrangler.toml.example  # Workers deployment config template
```

## Adding a lesson

1. Create `content-bundles/000N-your-lesson.json` (see existing files for format)
2. Run `make import`
3. Open `/lessons/your-lesson-slug`

Each lesson bundle includes sections (HTML content), questions (multiple choice or text),
and exercises (terminal practice). Answers are stored per-user at runtime — no answer keys
in the bundles.

## Development

```bash
make test      # run all tests (go test -race -shuffle=on)
make build     # compile native binaries
make import    # reload lesson content from JSON bundles
make serve     # start dev server on 127.0.0.1:4173
make worker    # build WASM binary for Cloudflare Workers
```

## Deploy to Cloudflare Workers

```bash
# 1. Copy the config template
cp wrangler.toml.example wrangler.toml

# 2. Edit wrangler.toml with your domain and D1 database ID

# 3. Create a D1 database (if you don't have one)
wrangler d1 create go-learn-db

# 4. Run migrations on D1
wrangler d1 execute go-learn-db --remote --file=internal/store/migrations/001_core.sql
wrangler d1 execute go-learn-db --remote --file=internal/store/migrations/002_users.sql

# 5. Import lesson content (dump from local SQLite, load into D1)

# 6. Build and deploy
make worker
wrangler deploy
```

Requires a Cloudflare account with Workers and D1 access. Migrations auto-apply
on Worker startup after the first manual run.

## Security

- **Passwords**: bcrypt hashed, never stored in plaintext
- **Sessions**: random 32-byte tokens, HttpOnly + Secure cookies
- **CSRF**: Origin header validation on all POST requests
- **CSP**: strict Content-Security-Policy (`script-src 'self'`, no CDN)
- **HSTS**: enabled over HTTPS (6 months + preload)
- **Rate limiting**: 10 POST requests per 30s window
- **Body size**: 1MB max on all requests

## License

MIT — see [LICENSE](LICENSE).
