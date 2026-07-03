// Package config loads the TOML configuration for hashcards serve.
package config

import (
	"github.com/BurntSushi/toml"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// DataConfig holds data storage settings.
type DataConfig struct {
	Root string `toml:"root"`
}

// FSRSSettings holds FSRS scheduling tuning parameters.
type FSRSSettings struct {
	TargetRecall float64 `toml:"target_recall"`
	MinInterval  float64 `toml:"min_interval"`
	MaxInterval  float64 `toml:"max_interval"`
}

// Config is the top-level structure parsed from the TOML file.
//
// Drill sessions are no longer configured here: "serve" derives one session
// per deck (plus a combined "All Decks" session) directly from the decks
// found under Data.Root. See internal/cmd/serve.
type Config struct {
	Server ServerConfig `toml:"server"`
	Data   DataConfig   `toml:"data"`
	FSRS   FSRSSettings `toml:"fsrs"`
}

// Load reads and parses the TOML config file at path, applying defaults.
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// Server defaults.
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3000
	}

	// Data defaults.
	if cfg.Data.Root == "" {
		cfg.Data.Root = "."
	}

	// FSRS defaults.
	if cfg.FSRS.TargetRecall == 0 {
		cfg.FSRS.TargetRecall = 0.9
	}
	if cfg.FSRS.MinInterval == 0 {
		cfg.FSRS.MinInterval = 1.0
	}
	if cfg.FSRS.MaxInterval == 0 {
		cfg.FSRS.MaxInterval = 256.0
	}

	return &cfg, nil
}
