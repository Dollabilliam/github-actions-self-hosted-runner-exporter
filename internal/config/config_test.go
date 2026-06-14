package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesDefaults(t *testing.T) {
	path := writeConfig(t, `{
	  "repositories": [
	    { "owner": "example-org", "name": "example-service" }
	  ]
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GitHub.BaseURL != "https://api.github.com" {
		t.Fatalf("BaseURL = %q", cfg.GitHub.BaseURL)
	}
	if cfg.GitHub.TokenEnv != "GITHUB_TOKEN" {
		t.Fatalf("TokenEnv = %q", cfg.GitHub.TokenEnv)
	}
	if cfg.GitHub.Timeout.Duration != 10*time.Second {
		t.Fatalf("Timeout = %s", cfg.GitHub.Timeout.Duration)
	}
	if cfg.Server.ListenAddress != ":9176" {
		t.Fatalf("ListenAddress = %q", cfg.Server.ListenAddress)
	}
	if cfg.Scrape.RefreshInterval.Duration != time.Minute {
		t.Fatalf("RefreshInterval = %s", cfg.Scrape.RefreshInterval.Duration)
	}
	if cfg.Scrape.RunsPerRepository != 20 {
		t.Fatalf("RunsPerRepository = %d", cfg.Scrape.RunsPerRepository)
	}
}

func TestLoadRejectsMissingRepositories(t *testing.T) {
	path := writeConfig(t, `{}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil")
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
