// main.go
package main

import (
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase"
	pbcmd "github.com/pocketbase/pocketbase/cmd"
	"github.com/spf13/cobra"

	"github.com/asano69/hatchcards/internal/cmd/check"
	"github.com/asano69/hatchcards/internal/cmd/orphans"
	"github.com/asano69/hatchcards/internal/cmd/serve"
	"github.com/asano69/hatchcards/internal/cmd/stats"
	"github.com/asano69/hatchcards/internal/config"

	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{HideStartBanner: true})

	// Registers "hatchcards migrate up/down/create/collections/history-sync"
	// for manual or CI-driven schema management. Automigrate is off because
	// the schema is defined purely in Go migration files (internal/migrations),
	// not edited through the PocketBase dashboard.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: false,
	})

	root := app.RootCmd
	root.Use = "hatchcards"
	root.Short = "A spaced-repetition flashcard tool"
	root.SilenceUsage = true
	root.Version = "0.1.0-beta.10"

	root.AddCommand(
		checkCmd(),
		statsCmd(app),
		orphansCmd(app),
		serveCmd(app),
		pbcmd.NewSuperuserCommand(app),
	)

	if err := app.Execute(); err != nil {
		os.Exit(1)
	}
}

// -------------------------------------------------------------------------
// check
// -------------------------------------------------------------------------
// check always runs against a disposable database (see db.OpenScratch), so
// it never touches the shared app or the real data directory.
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
func statsCmd(app *pocketbase.PocketBase) *cobra.Command {
	return &cobra.Command{
		Use:   "stats <collection-root>",
		Short: "Print card counts and review statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stats.Run(app, args[0], os.Stdout)
		},
	}
}

// -------------------------------------------------------------------------
// orphans
// -------------------------------------------------------------------------
func orphansCmd(app *pocketbase.PocketBase) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "orphans",
		Short:        "Commands relating to orphan cards",
		SilenceUsage: true,
	}
	cmd.AddCommand(orphansListCmd(app), orphansDeleteCmd(app))
	return cmd
}

func orphansListCmd(app *pocketbase.PocketBase) *cobra.Command {
	return &cobra.Command{
		Use:   "list <collection-root>",
		Short: "List the hashes of all orphan cards in the collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return orphans.List(app, args[0], os.Stdout)
		},
	}
}

func orphansDeleteCmd(app *pocketbase.PocketBase) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <collection-root>",
		Short: "Remove all orphan cards from the database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return orphans.Delete(app, args[0], os.Stdout)
		},
	}
}

// -------------------------------------------------------------------------
// serve
// -------------------------------------------------------------------------

func serveCmd(app *pocketbase.PocketBase) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the web server for all configured drill sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return serve.Run(app, cfg)
		},
	}
}
