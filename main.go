package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/asano69/hashcards/internal/cmd/check"
	"github.com/asano69/hashcards/internal/cmd/drill/server"
	"github.com/asano69/hashcards/internal/cmd/export"
	"github.com/asano69/hashcards/internal/cmd/orphans"
	"github.com/asano69/hashcards/internal/cmd/stats"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hashcards",
		Short: "A spaced-repetition flashcard tool",
		// Don't print usage on subcommand errors.
		SilenceUsage: true,
	}
	cmd.AddCommand(
		checkCmd(),
		statsCmd(),
		exportCmd(),
		orphansCmd(),
		drillCmd(),
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

// orphansCmd section of main.go — replace the existing orphansCmd() function
// with the version below, and update the import to use the rewritten package.

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
// drill
// -------------------------------------------------------------------------

func drillCmd() *cobra.Command {
	var (
		dbPath       string
		port         int
		seed         uint64
		staticDir    string
		cardLimit    int
		newCardLimit int
		fromDeck     string
		answerCtls   string
		burySiblings bool
	)
	cmd := &cobra.Command{
		Use:   "drill <collection-root>",
		Short: "Start the interactive drill web server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the current Unix timestamp as the default seed so each
			// session has a different shuffle without requiring user input.
			if !cmd.Flags().Changed("seed") {
				seed = uint64(time.Now().UnixNano())
			}
			if staticDir == "" {
				staticDir = defaultStaticDir()
			}

			cfg := server.Config{
				Root:           args[0],
				DBPath:         dbPath,
				Port:           port,
				Seed:           seed,
				StaticDir:      staticDir,
				Out:            os.Stdout,
				AnswerControls: server.AnswerControls(answerCtls),
				BurySiblings:   burySiblings,
			}

			// Only set pointer-based limits when the flag was explicitly provided,
			// so that omitting them means "no limit" rather than "limit=0".
			if cmd.Flags().Changed("card-limit") {
				cfg.CardLimit = &cardLimit
			}
			if cmd.Flags().Changed("new-card-limit") {
				cfg.NewCardLimit = &newCardLimit
			}
			if cmd.Flags().Changed("from-deck") {
				cfg.DeckFilter = &fromDeck
			}

			return server.Run(context.Background(), cfg)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "hashcards.db",
		"path to the performance database")
	cmd.Flags().IntVar(&port, "port", 8080,
		"TCP port for the drill web server")
	cmd.Flags().Uint64Var(&seed, "seed", 0,
		"PRNG seed for card shuffle (default: current time)")
	cmd.Flags().StringVar(&staticDir, "static", "",
		"path to the static assets directory (default: ./static)")
	cmd.Flags().IntVar(&cardLimit, "card-limit", 0,
		"Maximum number of cards to drill in a session (default: all due today)")
	cmd.Flags().IntVar(&newCardLimit, "new-card-limit", 0,
		"Maximum number of new cards to drill in a session")
	cmd.Flags().StringVar(&fromDeck, "from-deck", "",
		"Only drill cards from this deck")
	cmd.Flags().StringVar(&answerCtls, "answer-controls", "full",
		"Answer controls to show: full (Forgot/Hard/Good/Easy) or binary (Forgot/Good)")
	cmd.Flags().BoolVar(&burySiblings, "bury-siblings", true,
		"Bury sibling cloze cards within a session (default: true)")
	return cmd
}

// defaultStaticDir returns the "static" directory relative to the current
// working directory. This works correctly for both "go run ." and compiled
// binaries run from the project root.
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
