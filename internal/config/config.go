// Package config loads the TOML configuration for hashcards serve.
package config

import (
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// ServerConfig holds HTTP server and collection settings shared across all sessions.
type ServerConfig struct {
	Host         string  `toml:"host"`
	Port         int     `toml:"port"`
	DB           string  `toml:"db"`
	Root         string  `toml:"root"`
	TargetRecall float64 `toml:"target_recall"`
	MinInterval  float64 `toml:"min_interval"`
	MaxInterval  float64 `toml:"max_interval"`
}

// SessionConfig defines one drill session. Deck is the only required field;
// all routing and display names are derived from it automatically.
type SessionConfig struct {
	Deck           string `toml:"deck"`
	CardLimit      int    `toml:"card_limit"`
	NewCardLimit   int    `toml:"new_card_limit"`
	AnswerControls string `toml:"answer_controls"`
	BurySiblings   *bool  `toml:"bury_siblings"`

	// Derived fields computed from Deck during Load — not from TOML.
	Name     string
	Path     string
	FromDeck string
}

// Config is the top-level structure parsed from the TOML file.
type Config struct {
	Server   ServerConfig    `toml:"server"`
	Sessions []SessionConfig `toml:"session"`
}

// nonAlphanumRe matches runs of characters that are not lowercase letters or digits.
var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// deckToPath converts a deck name to a clean URL path segment.
// Empty string maps to "" (the root drill route /drill/).
// Other values are lowercased and non-alphanumeric runs become hyphens.
func deckToPath(deck string) string {
	if deck == "" {
		return ""
	}
	s := strings.ToLower(deck)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Load reads and parses the TOML config file at path, applying defaults and
// deriving computed fields from each session's Deck value.
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
	if cfg.Server.DB == "" {
		cfg.Server.DB = "hashcards.db"
	}
	if cfg.Server.Root == "" {
		cfg.Server.Root = "."
	}
	if cfg.Server.TargetRecall == 0 {
		cfg.Server.TargetRecall = 0.9
	}
	if cfg.Server.MinInterval == 0 {
		cfg.Server.MinInterval = 1.0
	}
	if cfg.Server.MaxInterval == 0 {
		cfg.Server.MaxInterval = 256.0
	}

	// Derive Name, Path, and FromDeck from each session's Deck field.
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		if s.Deck == "" {
			s.Name = "All Decks"
			s.Path = ""
			s.FromDeck = ""
		} else {
			s.Name = s.Deck
			s.Path = deckToPath(s.Deck)
			s.FromDeck = s.Deck
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
