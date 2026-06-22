package app

import (
	"os"
	"path/filepath"
)

// Config holds top-level application configuration.
type Config struct {
	Addr   string
	DBPath string
	Root   string
}

// Load reads configuration from the environment, using sensible defaults.
func Load() Config {
	root, _ := os.Getwd()
	port := os.Getenv("PORT")
	if port == "" {
		port = "4173"
	}
	return Config{
		Addr:   "127.0.0.1:" + port,
		DBPath: filepath.Join(root, "progress", "go-learn.db"),
		Root:   root,
	}
}
