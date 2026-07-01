// main.go
package main

import (
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase"
	pbcmd "github.com/pocketbase/pocketbase/cmd"
	"github.com/spf13/cobra"

	"github.com/asano69/hashcards/internal/cmd/check"
	"github.com/asano69/hashcards/internal/cmd/export"
	"github.com/asano69/hashcards/internal/cmd/orphans"
	"github.com/asano69/hashcards/internal/cmd/serve"
	"github.com/asano69/hashcards/internal/cmd/stats"
	"github.com/asano69/hashcards/internal/config"
)

func main() {
	// pocketbase.New() only builds the RootCmd; it does not register
	// PocketBase's own default commands (that only happens via Start()).
	// This lets us reuse PocketBase's cobra machinery (and its "superuser"
	// command, which operates on the same pb_data directory as "serve")
	// without pocketbase's built-in "serve" command shadowing ours.
	app := pocketbase.New()

	root := app.RootCmd
	root.Use = "hashcards"
	root.Short = "A spaced-repetition flashcard tool"
	root.SilenceUsage = true

	root.AddCommand(
		checkCmd(),
		statsCmd(),
		exportCmd(),
		orphansCmd(),
		serveCmd(),
		// PocketBase's built-in command for managing admin accounts.
		// It bootstraps its own app instance using the "--dir" flag
		// (default: "pb_data"), matching config.toml's [data].db default.
		pbcmd.NewSuperuserCommand(app),
	)

	// app.Execute() bootstraps the app and runs RootCmd with graceful
	// shutdown support, without registering PocketBase's default commands.
	if err := app.Execute(); err != nil {
		os.Exit(1)
	}
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
	var configPath string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web server for all configured drill sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load config %s: %w", configPath, err)
			}
			return serve.Run(cfg, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "config.toml",
		"path to the TOML config file")
	return cmd
}
