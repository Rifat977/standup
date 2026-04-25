package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/rifat977/standup/internal/ai"
	"github.com/rifat977/standup/internal/config"
	"github.com/rifat977/standup/internal/formatter"
	gitscan "github.com/rifat977/standup/internal/git"
	ghclient "github.com/rifat977/standup/internal/github"
	"github.com/rifat977/standup/internal/logx"
	"github.com/rifat977/standup/internal/ui"
)

var (
	flagSince   string
	flagAuthor  string
	flagModel   string
	flagNoAI    bool
	flagPlain   bool
	flagVerbose bool
)

func main() {
	root := &cobra.Command{
		Use:   "standup",
		Short: "Daily standup CLI — pulls commits, PRs, generates AI summary",
		RunE:  runTUI,
	}
	root.PersistentFlags().StringVar(&flagSince, "since", "", "time window (overrides config), e.g. 12h, 24h")
	root.PersistentFlags().StringVar(&flagAuthor, "author", "", "git author filter (overrides config)")
	root.PersistentFlags().StringVar(&flagModel, "model", "", "OpenAI model (overrides config)")
	root.PersistentFlags().BoolVar(&flagNoAI, "no-ai", false, "skip AI summary")
	root.PersistentFlags().BoolVar(&flagPlain, "plain", false, "plain-text output (for piping)")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "tee logs to stderr (debug level)")

	cobra.OnInitialize(initLogger)

	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create ~/.standup/config.yaml from the template",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Init()
			if err != nil && !errors.Is(err, config.ErrAlreadyExists) {
				return err
			}
			if errors.Is(err, config.ErrAlreadyExists) {
				fmt.Printf("config already exists at %s\n", p)
				return nil
			}
			fmt.Printf("wrote %s — edit to add tokens and repos\n", p)
			return nil
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print today's standup to stdout (no TUI)",
		RunE:  runShow,
	})

	if err := root.Execute(); err != nil {
		logx.Error("command failed: %v", err)
		fmt.Fprintln(os.Stderr, "error:", err)
		logx.Close()
		os.Exit(1)
	}
	logx.Close()
}

func initLogger() {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warn: cannot determine config dir:", err)
		return
	}
	if err := logx.Init(dir, flagVerbose); err != nil {
		fmt.Fprintln(os.Stderr, "warn: cannot open log file:", err)
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if flagSince != "" {
		cfg.Since = flagSince
	}
	if flagAuthor != "" {
		cfg.Author = flagAuthor
	}
	if flagModel != "" {
		cfg.OpenAI.Model = flagModel
	}
	return cfg, nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	m := ui.New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.Bind(p)
	_, err = p.Run()
	return err
}

func runShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()

	commits, _ := gitscan.Collect(cfg)
	prs, _ := ghclient.Collect(ctx, cfg)

	data := ai.Data{
		Commits: commits,
		PRs:     prs,
		Today:   "",
		Blocker: "",
	}

	if flagNoAI {
		fmt.Print(formatter.RenderRaw(data, pickFormat(cfg)))
		return nil
	}

	summary, err := ai.Summarize(ctx, cfg, data)
	if err != nil {
		return err
	}
	fmt.Print(formatter.RenderSummary(summary, pickFormat(cfg)))
	return nil
}

func pickFormat(cfg *config.Config) formatter.Format {
	if flagPlain {
		return formatter.Plain
	}
	switch cfg.Format {
	case "slack":
		return formatter.Slack
	case "plain":
		return formatter.Plain
	default:
		return formatter.Markdown
	}
}
