package app_test

import (
	"os"
	"strings"
	"testing"

	"example.com/go-learn/internal/app"
)

func TestConfigDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	cfg := app.Load()

	if !strings.Contains(cfg.Addr, ":4173") {
		t.Fatalf("want addr with :4173, got %q", cfg.Addr)
	}
	if !strings.HasSuffix(cfg.DBPath, "progress/go-learn.db") {
		t.Fatalf("want DBPath ending with progress/go-learn.db, got %q", cfg.DBPath)
	}
}

func TestConfigPortOverride(t *testing.T) {
	os.Setenv("PORT", "8080")
	defer os.Unsetenv("PORT")

	cfg := app.Load()
	if !strings.Contains(cfg.Addr, ":8080") {
		t.Fatalf("want addr with :8080, got %q", cfg.Addr)
	}
}
