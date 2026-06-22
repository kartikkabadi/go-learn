package app

import (
	"os"
	"path/filepath"
)

// Config holds top-level application configuration.
type Config struct {
	Addr      string
	DBPath    string
	Root      string
	BaseURL   string // canonical site URL for SEO (e.g. https://go-learn.dev); empty = derive per-request
	CookieKey []byte // HMAC key for signed session cookies; required for auth
}

// Load reads configuration from the environment, using sensible defaults.
func Load() Config {
	root, _ := os.Getwd()
	port := os.Getenv("PORT")
	if port == "" {
		port = "4173"
	}
	cfg := Config{
		Addr:    "127.0.0.1:" + port,
		DBPath:  filepath.Join(root, "progress", "go-learn.db"),
		Root:    root,
		BaseURL: os.Getenv("CANONICAL_BASE"),
	}
	if k := os.Getenv("COOKIE_KEY"); k != "" {
		cfg.CookieKey = []byte(k)
	}
	return cfg
}
