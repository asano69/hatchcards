// Package config loads the configuration for hashcards serve from
// environment variables.
package config

import (
	"os"
	"strconv"

	"github.com/asano69/hashcards/internal/errs"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string
	Port int
}

// DataConfig holds data storage settings.
type DataConfig struct {
	Root string
	// HooksDir is the directory of pre-installed post-sync hook scripts
	// (see internal/hook). It is read-only from the app's point of view;
	// an operator places executable scripts there ahead of time. A missing
	// directory is not an error: it simply means no hooks are available,
	// which is also the behavior of every installation predating this
	// feature.
	HooksDir string
}

// FSRSSettings holds FSRS scheduling tuning parameters.
type FSRSSettings struct {
	TargetRecall float64
	MinInterval  float64
	MaxInterval  float64
}

// Config is the top-level configuration, loaded from environment variables.
//
// Drill sessions are not configured here: "serve" derives one session
// per deck (plus a combined "All Decks" session) directly from the decks
// found under Data.Root. See internal/cmd/serve.
type Config struct {
	Server ServerConfig
	Data   DataConfig
	FSRS   FSRSSettings
}

// Load reads configuration from environment variables, applying defaults
// for any variable that is unset.
//
// Recognised variables:
//
//	SERVER_HOST         default "0.0.0.0"
//	SERVER_PORT         default 3000
//	DATA_ROOT           default "."
//	HOOKS_DIR           default "./hooks"
//	FSRS_TARGET_RECALL  default 0.9
//	FSRS_MIN_INTERVAL   default 1.0
//	FSRS_MAX_INTERVAL   default 256.0
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: envString("SERVER_HOST", "0.0.0.0"),
			Port: 3000,
		},
		Data: DataConfig{
			Root:     envString("DATA_ROOT", "."),
			HooksDir: envString("HOOKS_DIR", "./hooks"),
		},
		FSRS: FSRSSettings{
			TargetRecall: 0.9,
			MinInterval:  1.0,
			MaxInterval:  256.0,
		},
	}

	port, err := envInt("SERVER_PORT", cfg.Server.Port)
	if err != nil {
		return nil, err
	}
	cfg.Server.Port = port

	targetRecall, err := envFloat("FSRS_TARGET_RECALL", cfg.FSRS.TargetRecall)
	if err != nil {
		return nil, err
	}
	cfg.FSRS.TargetRecall = targetRecall

	minInterval, err := envFloat("FSRS_MIN_INTERVAL", cfg.FSRS.MinInterval)
	if err != nil {
		return nil, err
	}
	cfg.FSRS.MinInterval = minInterval

	maxInterval, err := envFloat("FSRS_MAX_INTERVAL", cfg.FSRS.MaxInterval)
	if err != nil {
		return nil, err
	}
	cfg.FSRS.MaxInterval = maxInterval

	return cfg, nil
}

// envString returns the value of the environment variable key, or fallback
// if it is unset or empty.
func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// envInt returns the integer value of the environment variable key, or
// fallback if it is unset.
func envInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, errs.Newf("invalid %s: %v", key, err)
	}
	return n, nil
}

// envFloat returns the float64 value of the environment variable key, or
// fallback if it is unset.
func envFloat(key string, fallback float64) (float64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, errs.Newf("invalid %s: %v", key, err)
	}
	return f, nil
}
