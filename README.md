# go-learn

[![CI](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml/badge.svg)](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev)
[![Tailwind CSS](https://img.shields.io/badge/CSS-Tailwind_v4-38BDF8.svg)](#)

A free, interactive Go programming course with lessons, quizzes, and hands-on exercises.
Server-rendered with Go + HTMX. Tailwind CSS (compiled, no JS framework). Deployed on Cloudflare Workers with D1.

**→ [go-learn.kartikkabadi.com](https://go-learn.kartikkabadi.com)**

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

## Tech

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, `net/http`, `html/template` |
| Frontend | HTMX 2.x, Tailwind CSS v4, shadcn-inspired tokens |
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
├── web/static/            # input.css (source), theme.css (compiled), HTMX, OG image
└── wrangler.toml.example  # Workers deployment config template
```

## Development

```bash
make import    # load content bundles into SQLite
make css       # compile Tailwind (web/static/input.css → theme.css)
make serve     # http://127.0.0.1:4173/
```

Visual design rules live in [`DESIGN.md`](DESIGN.md). Run `make css-watch` while editing templates or CSS.

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
