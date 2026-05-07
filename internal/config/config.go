package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SourceConfig holds settings for a specific content source.
type SourceConfig struct {
	Enabled bool     `yaml:"enabled"`
	Sort    string   `yaml:"sort"`
	Targets []string `yaml:"targets,omitempty"` // subreddits, lemmy instances, etc.
}

// BrowserConfig holds settings for the browser.
type BrowserConfig struct {
	Command   string `yaml:"command"`
	Arguments string `yaml:"arguments"`
}

// Config holds the application configuration.
type Config struct {
	Sources map[string]SourceConfig `yaml:"sources"`
	Browser *BrowserConfig          `yaml:"browser"`
}

// Valid sort options per source
var ValidSorts = map[string][]string{
	"hn":       {"top", "new", "best"},
	"reddit":   {"hot", "new", "top", "rising"},
	"lobsters": {"hottest", "newest"},
	"lemmy":    {"Hot", "New", "Active", "TopDay", "TopWeek", "TopAll"},
	"devto":    {"default", "latest", "top", "rising"},
	"rss":      {"default"},
}

// Registry for backwards compatibility with existing UI code if needed
var (
	RedditSorts = ValidSorts["reddit"]
	HNSorts     = ValidSorts["hn"]
)

// DefaultSubreddits is the default list of subreddits to fetch.
var DefaultSubreddits = []string{"programming", "linux", "opencodecli", "claudecode"}

// DefaultRSSFeeds is the default list of RSS feeds to fetch.
var DefaultRSSFeeds = []string{"https://lwn.net/headlines/rss", "https://feeds.arstechnica.com/arstechnica/technology-lab"}

// Path returns the path to the config file.
func Path() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "(unknown)"
	}
	return filepath.Join(configDir, "technews-tui", "config.yaml")
}

// Load reads the config from ~/.config/technews-tui/config.yaml.
// If the file doesn't exist, it creates one with defaults.
func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return defaultConfig(), nil
	}

	dir := filepath.Join(configDir, "technews-tui")
	path := filepath.Join(dir, "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist — create default
		cfg := defaultConfig()
		_ = writeDefault(dir, path, cfg)
		return cfg, nil
	}

	// Try to detect old flat format first
	var oldFormat struct {
		Subreddits []string `yaml:"subreddits"`
		RedditSort string   `yaml:"reddit_sort"`
		HNSort     string   `yaml:"hn_sort"`
	}
	if err := yaml.Unmarshal(data, &oldFormat); err == nil && len(oldFormat.Subreddits) > 0 {
		// Migration
		cfg := defaultConfig()
		cfg.Sources["reddit"] = SourceConfig{
			Enabled: true,
			Sort:    oldFormat.RedditSort,
			Targets: oldFormat.Subreddits,
		}
		cfg.Sources["hn"] = SourceConfig{
			Enabled: true,
			Sort:    oldFormat.HNSort,
		}
		cfg.ValidateSort()
		_ = Save(cfg) // Update file to new format
		return cfg, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), nil
	}

	if len(cfg.Sources) == 0 {
		cfg = *defaultConfig()
	}
	cfg.ValidateSort()
	return &cfg, nil
}

// Save writes the config back to ~/.config/technews-tui/config.yaml.
func Save(cfg *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(configDir, "technews-tui")
	path := filepath.Join(dir, "config.yaml")
	return writeDefault(dir, path, cfg)
}

func defaultConfig() *Config {
	return &Config{
		Sources: map[string]SourceConfig{
			"hn": {
				Enabled: true,
				Sort:    "top",
			},
			"reddit": {
				Enabled: true,
				Sort:    "hot",
				Targets: DefaultSubreddits,
			},
			"lobsters": {
				Enabled: true,
				Sort:    "hottest",
			},
			"lemmy": {
				Enabled: true,
				Sort:    "Hot",
				Targets: []string{"lemmy.ml", "programming.dev"},
			},
			"devto": {
				Enabled: true,
				Sort:    "default",
			},
			"rss": {
				Enabled: true,
				Sort:    "default",
				Targets: DefaultRSSFeeds,
			},
		},
	}
}

// ValidateSort ensures sort values are valid, resetting to defaults if not.
func (c *Config) ValidateSort() {
	for id, sc := range c.Sources {
		valid, ok := ValidSorts[id]
		if !ok {
			continue
		}
		if !contains(valid, sc.Sort) {
			sc.Sort = valid[0]
			c.Sources[id] = sc
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func writeDefault(dir, path string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := []byte("# technews-tui configuration\n# Manage your sources and preferences below.\n\n")
	return os.WriteFile(path, append(header, data...), 0o644)
}
