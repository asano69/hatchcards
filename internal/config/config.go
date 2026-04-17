// Package config loads the TOML configuration for hashcards serve.
package config

import (
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// DataConfig holds data storage settings.
type DataConfig struct {
	DB   string `toml:"db"`
	Root string `toml:"root"`
}

// FSRSSettings holds FSRS scheduling tuning parameters.
type FSRSSettings struct {
	TargetRecall float64 `toml:"target_recall"`
	MinInterval  float64 `toml:"min_interval"`
	MaxInterval  float64 `toml:"max_interval"`
}

// SessionConfig defines one drill session. Deck is the deck filter;
// Title overrides the display name shown in the index.
type SessionConfig struct {
	Title          string `toml:"title"`
	Deck           string `toml:"deck"`
	CardLimit      int    `toml:"card_limit"`
	NewCardLimit   int    `toml:"new_card_limit"`
	AnswerControls string `toml:"answer_controls"`
	BurySiblings   *bool  `toml:"bury_siblings"`

	// Derived fields computed during Load — not from TOML.
	Name     string
	Path     string
	FromDeck string
}

// Config is the top-level structure parsed from the TOML file.
type Config struct {
	Server   ServerConfig    `toml:"server"`
	Data     DataConfig      `toml:"data"`
	FSRS     FSRSSettings    `toml:"fsrs"`
	Sessions []SessionConfig `toml:"session"`
}

// nonAlphanumRe matches runs of characters that are not lowercase letters or digits.
var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// deckToPath converts a deck name to a clean URL path segment.
// Empty string maps to "" (the root drill route /drill/).
func deckToPath(deck string) string {
	if deck == "" {
		return ""
	}
	s := strings.ToLower(deck)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Load reads and parses the TOML config file at path, applying defaults and
// deriving computed fields from each session's Deck and Title values.
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
	if cfg.Data.DB == "" {
		cfg.Data.DB = "hashcards.db"
	}
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

	// Derive Name, Path, and FromDeck from each session's fields.
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		s.Path = deckToPath(s.Deck)
		s.FromDeck = s.Deck
		switch {
		case s.Title != "":
			s.Name = s.Title
		case s.Deck != "":
			s.Name = s.Deck
		default:
			s.Name = "All Decks"
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
