package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default configuration values.
const (
	DefaultConfigFile      = "config.yaml"
	DefaultBrainPath       = "./brain"
	DefaultSessionsPath    = "./sessions"
	DefaultMaxContextFiles = 10
	DefaultListenAddr      = "0.0.0.0:8080"
	DefaultProviderTimeout = 120 // seconds
)

// Config is the top-level application configuration.
type Config struct {
	Provider     string                    `yaml:"provider"`
	Providers    map[string]ProviderConfig `yaml:"providers"`
	Brain        BrainConfig               `yaml:"brain"`
	SessionsPath string                    `yaml:"sessions_path"`
	Server       ServerConfig              `yaml:"server"`
	Google       GoogleConfig              `yaml:"google"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

// GoogleConfig holds Google API settings.
type GoogleConfig struct {
	ServiceAccountPath string `yaml:"service_account_path"`
}

// ProviderConfig holds per-provider settings.
type ProviderConfig struct {
	Auth       string `yaml:"auth"`
	APIKey     string `yaml:"api_key"`
	Model      string `yaml:"model"`
	BaseURL    string `yaml:"base_url"`
	TimeoutSec int    `yaml:"timeout_sec"`
}

// BrainConfig holds brain storage settings.
type BrainConfig struct {
	Path            string     `yaml:"path"`
	MaxContextFiles int        `yaml:"max_context_files"`
	Sync            SyncConfig `yaml:"sync"`
}

// SyncConfig holds brain sync settings.
type SyncConfig struct {
	Git   GitSyncConfig   `yaml:"git"`
	Cloud CloudSyncConfig `yaml:"cloud"`
}

// GitSyncConfig holds git-based sync settings.
type GitSyncConfig struct {
	Enabled bool   `yaml:"enabled"`
	Remote  string `yaml:"remote"`
	Branch  string `yaml:"branch"`
}

// CloudSyncConfig holds cloud backup settings.
type CloudSyncConfig struct {
	Backend    string           `yaml:"backend"`
	GDrive     GDriveConfig     `yaml:"gdrive"`
	VercelBlob VercelBlobConfig `yaml:"vercel_blob"`
}

// GDriveConfig holds Google Drive backup settings.
type GDriveConfig struct {
	FolderID string `yaml:"folder_id"`
}

// VercelBlobConfig holds Vercel Blob backup settings.
type VercelBlobConfig struct {
	Token string `yaml:"token"`
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// Load reads and parses a config file, expanding ${ENV_VAR} references.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var missingVars []string
	expanded := envVarPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		varName := string(envVarPattern.FindSubmatch(match)[1])
		val := os.Getenv(varName)
		if val == "" {
			missingVars = append(missingVars, varName)
		}
		return []byte(val)
	})
	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing environment variables: %s", strings.Join(missingVars, ", "))
	}

	cfg := Defaults()
	if err := yaml.Unmarshal(expanded, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// Defaults returns a Config with sensible defaults and no provider.
func Defaults() *Config {
	return &Config{
		Brain: BrainConfig{
			Path:            DefaultBrainPath,
			MaxContextFiles: DefaultMaxContextFiles,
		},
		SessionsPath: DefaultSessionsPath,
		Server: ServerConfig{
			ListenAddr: DefaultListenAddr,
		},
	}
}
