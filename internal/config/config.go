package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed example.yaml
var exampleYAML []byte

type Config struct {
	Author   string   `yaml:"author"`
	Since    string   `yaml:"since"`
	ScanDirs []string `yaml:"scan_dirs"`
	GitHub   GitHub   `yaml:"github"`
	OpenAI   OpenAI   `yaml:"openai"`
	Slack    Slack    `yaml:"slack"`
	Format   string   `yaml:"format"`

	path string `yaml:"-"`
}

type GitHub struct {
	Token string   `yaml:"token"`
	Repos []string `yaml:"repos"`
}

type OpenAI struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

type Slack struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

func Default() *Config {
	return &Config{
		Since:    "12h",
		ScanDirs: []string{"~/projects"},
		OpenAI:   OpenAI{Model: "gpt-4o", MaxTokens: 500},
		Slack:    Slack{Channel: "#standup"},
		Format:   "markdown",
	}
}

// Path returns the canonical config path: ~/.standup/config.yaml.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".standup", "config.yaml"), nil
}

// Dir returns ~/.standup.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".standup"), nil
}

// Load reads ~/.standup/config.yaml, applies env overrides, and expands ~ in scan_dirs.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config not found at %s — run `standup init`", p)
		}
		return nil, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	cfg.path = p
	cfg.applyEnv()
	cfg.expandPaths()
	return cfg, nil
}

// Save writes the current config back to its source path.
func (c *Config) Save() error {
	if c.path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		c.path = p
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o600)
}

// Init creates ~/.standup/ and writes the example config if it does not exist.
func Init() (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		return p, fs_exists
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, exampleYAML, 0o600); err != nil {
		return "", err
	}
	return p, nil
}

var fs_exists = errors.New("config already exists")

// ErrAlreadyExists is returned by Init when the config file is already present.
var ErrAlreadyExists = fs_exists

func (c *Config) applyEnv() {
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		c.OpenAI.APIKey = v
	}
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		c.GitHub.Token = v
	}
	if v := os.Getenv("SLACK_WEBHOOK_URL"); v != "" {
		c.Slack.WebhookURL = v
	}
}

func (c *Config) expandPaths() {
	for i, d := range c.ScanDirs {
		c.ScanDirs[i] = ExpandHome(d)
	}
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}
