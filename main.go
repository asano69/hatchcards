package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/asano69/hashcards/internal/cmd/check"
	"github.com/asano69/hashcards/internal/cmd/export"
	"github.com/asano69/hashcards/internal/cmd/orphans"
	"github.com/asano69/hashcards/internal/cmd/serve"
	"github.com/asano69/hashcards/internal/cmd/stats"
	"github.com/asano69/hashcards/internal/config"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "hashcards",
		Short:        "A spaced-repetition flashcard tool",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		checkCmd(),
		statsCmd(),
		exportCmd(),
		orphansCmd(),
		serveCmd(),
	)
	return cmd
}

// -------------------------------------------------------------------------
// check
// -------------------------------------------------------------------------

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <collection-root>",
		Short: "Parse all deck files and validate media references",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := check.Run(args[0], os.Stdout)
			if err != nil {
				return err
			}
			if !result.OK() {
				return fmt.Errorf("%d error(s) found", len(result.Errors))
			}
			fmt.Fprintf(os.Stdout, "OK: %d card(s) checked.\n", result.CardCount)
			return nil
		},
	}
}

// -------------------------------------------------------------------------
// stats
// -------------------------------------------------------------------------

func statsCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "stats <collection-root>",
		Short: "Print card counts and review statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stats.Run(args[0], dbPath, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "hashcards.db",
		"path to the performance database")
	return cmd
}

// -------------------------------------------------------------------------
// export
// -------------------------------------------------------------------------

func exportCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export review history as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return export.Run(dbPath, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "hashcards.db",
		"path to the performance database")
	return cmd
}

// -------------------------------------------------------------------------
// orphans
// -------------------------------------------------------------------------

func orphansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "orphans",
		Short:        "Commands relating to orphan cards",
		SilenceUsage: true,
	}
	cmd.AddCommand(orphansListCmd(), orphansDeleteCmd())
	return cmd
}

func orphansListCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "list <collection-root>",
		Short: "List the hashes of all orphan cards in the collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return orphans.List(args[0], dbPath, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "hashcards.db",
		"path to the performance database")
	return cmd
}

func orphansDeleteCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "delete <collection-root>",
		Short: "Remove all orphan cards from the database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return orphans.Delete(args[0], dbPath, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "hashcards.db",
		"path to the performance database")
	return cmd
}

// -------------------------------------------------------------------------
// serve
// -------------------------------------------------------------------------

func serveCmd() *cobra.Command {
	var (
		configPath string
		staticDir  string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web server for all configured drill sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load config %s: %w", configPath, err)
			}
			if staticDir == "" {
				staticDir = defaultStaticDir()
			}
			return serve.Run(cfg, staticDir, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "config.toml",
		"path to the TOML config file")
	cmd.Flags().StringVar(&staticDir, "static", "",
		"path to the static assets directory (default: ./static)")
	return cmd
}

// defaultStaticDir returns the "static" directory relative to the binary or cwd.
func defaultStaticDir() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "static")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "static"
}
