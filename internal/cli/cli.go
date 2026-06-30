package cli

import (
	"fmt"
	"io"

	"github.com/alecthomas/kong"

	"github.com/asano69/hashcards/internal/cmd/check"
	"github.com/asano69/hashcards/internal/cmd/export"
	"github.com/asano69/hashcards/internal/cmd/orphans"
	"github.com/asano69/hashcards/internal/cmd/serve"
	"github.com/asano69/hashcards/internal/cmd/stats"
	"github.com/asano69/hashcards/internal/config"
)

const description = "A spaced-repetition flashcard tool"

type CLI struct {
	Check   CheckCmd   `cmd:"" help:"Parse all deck files and validate media references."`
	Stats   StatsCmd   `cmd:"" help:"Print card counts and review statistics."`
	Export  ExportCmd  `cmd:"" help:"Export review history as JSON."`
	Orphans OrphansCmd `cmd:"" help:"Commands relating to orphan cards."`
	Serve   ServeCmd   `cmd:"" help:"Start the web server for all configured drill sessions."`
}

type CheckCmd struct {
	CollectionRoot string `arg:"" name:"collection-root" help:"Path to the collection root."`
}

func (c *CheckCmd) Run(out io.Writer) error {
	result, err := check.Run(c.CollectionRoot, out)
	if err != nil {
		return err
	}
	if !result.OK() {
		return fmt.Errorf("%d error(s) found", len(result.Errors))
	}
	fmt.Fprintf(out, "OK: %d card(s) checked.\n", result.CardCount)
	return nil
}

type StatsCmd struct {
	CollectionRoot string `arg:"" name:"collection-root" help:"Path to the collection root."`
	DB             string `help:"Path to the performance database." default:"hashcards.db"`
}

func (c *StatsCmd) Run(out io.Writer) error { return stats.Run(c.CollectionRoot, c.DB, out) }

type ExportCmd struct {
	DB string `help:"Path to the performance database." default:"hashcards.db"`
}

func (c *ExportCmd) Run(out io.Writer) error { return export.Run(c.DB, out) }

type OrphansCmd struct {
	List   OrphansListCmd   `cmd:"" help:"List the hashes of all orphan cards in the collection."`
	Delete OrphansDeleteCmd `cmd:"" help:"Remove all orphan cards from the database."`
}

type OrphansListCmd struct {
	CollectionRoot string `arg:"" name:"collection-root" help:"Path to the collection root."`
	DB             string `help:"Path to the performance database." default:"hashcards.db"`
}

func (c *OrphansListCmd) Run(out io.Writer) error { return orphans.List(c.CollectionRoot, c.DB, out) }

type OrphansDeleteCmd struct {
	CollectionRoot string `arg:"" name:"collection-root" help:"Path to the collection root."`
	DB             string `help:"Path to the performance database." default:"hashcards.db"`
}

func (c *OrphansDeleteCmd) Run(out io.Writer) error {
	return orphans.Delete(c.CollectionRoot, c.DB, out)
}

type ServeCmd struct {
	Config string `help:"Path to the TOML config file." default:"config.toml"`
}

func (c *ServeCmd) Run(out io.Writer) error {
	cfg, err := config.Load(c.Config)
	if err != nil {
		return fmt.Errorf("load config %s: %w", c.Config, err)
	}
	return serve.Run(cfg, out)
}

func Run(args []string, stdout, stderr io.Writer) error {
	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("hashcards"),
		kong.Description(description),
		kong.Writers(stdout, stderr),
		kong.BindTo(stdout, (*io.Writer)(nil)),
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	return ctx.Run()
}
