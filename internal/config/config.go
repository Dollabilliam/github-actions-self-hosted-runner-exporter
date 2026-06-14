package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("duration must be a string: %w", err)
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	d.Duration = parsed
	return nil
}

type Config struct {
	GitHub       GitHubConfig `json:"github"`
	Server       ServerConfig `json:"server"`
	Scrape       ScrapeConfig `json:"scrape"`
	Repositories []Repository `json:"repositories"`
}

type GitHubConfig struct {
	BaseURL  string   `json:"base_url"`
	TokenEnv string   `json:"token_env"`
	Timeout  Duration `json:"timeout"`
}

type ServerConfig struct {
	ListenAddress string `json:"listen_address"`
}

type ScrapeConfig struct {
	RefreshInterval   Duration `json:"refresh_interval"`
	RunsPerRepository int      `json:"runs_per_repository"`
}

type Repository struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		GitHub: GitHubConfig{
			BaseURL:  "https://api.github.com",
			TokenEnv: "GITHUB_TOKEN",
			Timeout:  Duration{Duration: 10 * time.Second},
		},
		Server: ServerConfig{
			ListenAddress: ":9176",
		},
		Scrape: ScrapeConfig{
			RefreshInterval:   Duration{Duration: 60 * time.Second},
			RunsPerRepository: 20,
		},
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.GitHub.BaseURL == "" {
		return errors.New("github.base_url is required")
	}
	if c.GitHub.TokenEnv == "" {
		return errors.New("github.token_env is required")
	}
	if c.GitHub.Timeout.Duration <= 0 {
		return errors.New("github.timeout must be greater than zero")
	}
	if c.Server.ListenAddress == "" {
		return errors.New("server.listen_address is required")
	}
	if c.Scrape.RefreshInterval.Duration <= 0 {
		return errors.New("scrape.refresh_interval must be greater than zero")
	}
	if c.Scrape.RunsPerRepository <= 0 {
		return errors.New("scrape.runs_per_repository must be greater than zero")
	}
	if len(c.Repositories) == 0 {
		return errors.New("repositories must include at least one repository")
	}

	for i, repo := range c.Repositories {
		if repo.Owner == "" {
			return fmt.Errorf("repositories[%d].owner is required", i)
		}
		if repo.Name == "" {
			return fmt.Errorf("repositories[%d].name is required", i)
		}
	}

	return nil
}
