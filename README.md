# go-learn

[![CI](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml/badge.svg)](https://github.com/kartikkabadi/go-learn/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev)
[![Zero JS framework](https://img.shields.io/badge/JS_framework-zero-00ADD8.svg)](#)

A free, interactive Go programming course with lessons, quizzes, and hands-on exercises.
Server-rendered with Go + HTMX. No JavaScript framework, no build step.

**→ [go-learn.kartikkabadi.com](https://go-learn.kartikkabadi.com)**

## Run locally

```bash
git clone https://github.com/kartikkabadi/go-learn.git
cd go-learn
make import   # load lessons into SQLite
make serve    # open http://127.0.0.1:4173/
```

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
| Frontend | HTMX 2.x, vanilla CSS, no build step |
| Database | SQLite (dev) / Cloudflare D1 (prod) |
| Deploy | Cloudflare Workers (WASM) |
| Auth | bcrypt + session cookies |

## Development

```bash
make test      # run all tests
make import    # reload lesson content from JSON bundles
make serve     # start dev server
```

See [CONTRIBUTING.md](CONTRIBUTING.md) to add a lesson.

## License

MIT — see [LICENSE](LICENSE).
