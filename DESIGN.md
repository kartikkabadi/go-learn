# go-learn Design System

Visual SSOT for the frontend revamp. Go server-rendered templates + Tailwind CSS + shadcn-inspired tokens. No React.

## Stack

| Layer | Choice |
|-------|--------|
| Templates | Go `html/template` |
| Interactivity | HTMX 2.x |
| Styling | Tailwind CSS v4 (compiled from `web/static/input.css`) |
| Components | Tailwind `@layer components` + template partials |
| Fonts | System UI stack (self-host Geist later if needed) |

## Design direction

**Dev-education minimal** — clean shadcn/new-york aesthetic, Go-teal primary, readable lesson content, subtle depth (borders + light shadows). Not a marketing splash page; not generic AI slop.

### Density: 3/5
Comfortable reading width (`max-w-3xl`), clear section spacing, cards for grouped content.

### Motion: 2/5
CSS transitions only. HTMX swaps, progress bars, hover states. No Framer Motion, no WebGL.

## Tokens

Defined in `web/static/input.css`. Primary uses Go-adjacent teal (`oklch` hue ~210).

| Token | Use |
|-------|-----|
| `--background` / `--foreground` | Page canvas and body text |
| `--card` | Cards, inputs, elevated surfaces |
| `--primary` | Links, CTAs, progress, quiz borders |
| `--muted` / `--muted-foreground` | Secondary text, table headers |
| `--success` / `--destructive` | Quiz correct/wrong feedback |
| `--radius` | `0.5rem` — shadcn default |

Dark mode follows `prefers-color-scheme` (no toggle yet).

## Typography

- **UI / body:** `ui-sans-serif, system-ui, …`
- **Code:** `ui-monospace, SF Mono, Menlo, Consolas`

## Component classes

Semantic classes in `@layer components` (maps to shadcn patterns):

| Class | Role |
|-------|------|
| `.btn` | Secondary/outline button |
| `.btn-primary` | Primary CTA |
| `.btn-large` | Hero / signup CTAs |
| `.card` | Content container |
| `.stat` / `.stat-row` | Dashboard metrics |
| `.progress-bar` | Lesson completion |
| `.quiz-block` | Interactive quiz container |
| `.auth-form` | Login/signup fields |
| `.hero` | Landing hero section |

## Template layout

```
partials/head.html      — meta, OG, CSS, optional HTMX/JSON-LD
partials/page_start.html — <body>, nav, <main>
partials/page_end.html   — close tags
partials/nav.html        — site navigation
```

## Bans

- Inter as primary font (use system stack or Geist if self-hosted)
- Pure `#000` / `#fff` backgrounds in dark/light themes
- React component libraries (shadcn/ui, Cult UI, Skiper, etc.) — port aesthetics only
- CDN scripts/styles (CSP: `script-src 'self'`, `style-src 'self'`)
- Gradient text on headings
- Purple-on-white "AI startup" palette

## Inspiration libraries (reference only)

Port CSS/HTML patterns, do not install:

- styleui.dev — landing layouts
- cult-ui / skiper-ui — button/card polish
- dotmatrix — HTMX loading indicators (CSS `@keyframes`)
- componentry — scroll/card patterns (CSS-only subset)

Skip: metal-fx (WebGL + React), balloons-js.

## Build

```bash
npm run css:build   # compile input.css → theme.css
make serve          # run Go server
```

CI runs `npm run css:build` before tests.
