package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the entire application configuration.
// It mirrors the structure of the config.toml file.
// We use `map[string]Model` for quick lookups by model alias.


type Config struct {
	LogLevel   string             `toml:"log_level"`
	Server     ServerConfig       `toml:"server"`
	Models     map[string]Model   `toml:"-"` // Populated after parsing
	RawModels  []Model            `toml:"models"` // Used for initial parsing
}

// ServerConfig holds server-specific configuration settings.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// Model represents a model alias mapping to a target provider.
type Model struct {
	Alias  string       `toml:"alias"`
	Target TargetConfig `toml:"target"`
	Type   string       `toml:"type"`
}

// TargetConfig holds the target provider details.
type TargetConfig struct {
	URL    string `toml:"url"`
	Model  string `toml:"model"`
	APIKey string `toml:"api_key"`
}

// Load reads the configuration from the specified file path,
// parses it, and returns a populated Config struct.


func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}

	// Convert the slice of models into a map for efficient access by alias.
	cfg.Models = make(map[string]Model)
	for _, model := range cfg.RawModels {
		// Resolve environment variables in API keys
		if envVar, found := strings.CutPrefix(model.Target.APIKey, "env:"); found {
			if envValue := os.Getenv(envVar); envValue != "" {
				model.Target.APIKey = envValue
			}
		}
		cfg.Models[model.Alias] = model
	}
	// We don't need the raw slice anymore.
	cfg.RawModels = nil

	// Set default server configuration if not provided
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	return &cfg, nil
}

// Address returns the server address in the format "host:port".
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
