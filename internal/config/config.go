// Package config loads the TOML configuration for hashcards serve.
package config

import "github.com/BurntSushi/toml"

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `toml:"port"`
}

// SessionConfig defines one drill session exposed under /drill/<path>.
type SessionConfig struct {
	Name           string `toml:"name"`
	Path           string `toml:"path"`
	Root           string `toml:"root"`
	DB             string `toml:"db"`
	FromDeck       string `toml:"from_deck"`
	CardLimit      int    `toml:"card_limit"`
	NewCardLimit   int    `toml:"new_card_limit"`
	AnswerControls string `toml:"answer_controls"`
	BurySiblings   *bool  `toml:"bury_siblings"`
}

// Config is the top-level structure parsed from the TOML file.
type Config struct {
	Server   ServerConfig    `toml:"server"`
	Sessions []SessionConfig `toml:"session"`
}

// Load reads and parses the TOML config file at path.
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3000
	}
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		if s.DB == "" {
			s.DB = "hashcards.db"
		}
		if s.AnswerControls == "" {
			s.AnswerControls = "full"
		}
		if s.BurySiblings == nil {
			t := true
			s.BurySiblings = &t
		}
	}
	return &cfg, nil
}
