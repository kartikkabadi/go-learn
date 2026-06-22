# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | ✓         |

## Reporting a Vulnerability

If you discover a security vulnerability, please open a GitHub
Security Advisory rather than a public issue. Go to
**Security → Advisories → New advisory** on the repo.

We will respond within 48 hours and work to resolve confirmed issues promptly.

## Security Features

- **Passwords**: bcrypt hashed, never stored in plaintext
- **Sessions**: random 32-byte tokens, HttpOnly + Secure + SameSite cookies
- **CSRF**: Origin header validation on all POST requests
- **CSP**: strict Content-Security-Policy (`script-src 'self'`, no CDN)
- **HSTS**: enabled over HTTPS (6 months + preload)
- **Rate limiting**: 10 POST requests per 30s window per IP
- **Body size**: 1MB max on all requests
- **Security headers**: X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy
