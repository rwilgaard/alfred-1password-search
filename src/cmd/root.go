package cmd

import (
	"fmt"
	"os"

	"github.com/rwilgaard/go-alfredutils/v2/alfred"
	"github.com/rwilgaard/go-alfredutils/v2/alfred/update"
	"github.com/spf13/cobra"
	"go.deanishe.net/fuzzy"
)

type workflowConfig struct {
	ModifierReturn        string `env:"modifier_return"`
	ModifierCmd           string `env:"modifier_cmd"`
	ModifierOpt           string `env:"modifier_opt"`
	ModifierCtrl          string `env:"modifier_ctrl"`
	ModifierShift         string `env:"modifier_shift"`
	AllowedSymbols        string `env:"allowed_symbols"`
	FuzzySearch           bool   `env:"fuzzy_search"`
	ClipboardTimeout      int    `env:"clipboard_timeout"` // seconds; 0 never clears
	VaultName             string `env:"vault_name"`
	SelectedVaults        string `env:"selected_vaults"`
	SelectedVaultsKeyword string `env:"selected_vaults_keyword"`
	ListQuery             string `env:"list_query"`
	ItemID                string `env:"item_id"`
}

const repo = "rwilgaard/alfred-1password-search"

const (
	itemsCacheKey = "items.json"
	syncJobName   = "sync"
)

var (
	wf      *alfred.Workflow
	updater *update.Updater
	cfg     = &workflowConfig{}

	rootCmd = &cobra.Command{
		Use:           "alfred-1password-search",
		Short:         "Alfred workflow for searching 1Password vault items",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func Execute() {
	var err error
	wf, err = alfred.New(
		alfred.WithMaxResults(25),
		alfred.WithUsage(),
		alfred.WithFuzzy(
			fuzzy.AdjacencyBonus(10.0),
			fuzzy.LeadingLetterPenalty(-0.1),
			fuzzy.MaxLeadingLetterPenalty(-3.0),
			fuzzy.UnmatchedLetterPenalty(-0.5),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	updater = update.GitHub(wf, repo)
	wf.Run(run)
}

func run() {
	if err := wf.Config(cfg); err != nil {
		wf.FatalError(err)
	}
	if err := updater.Run(); err != nil {
		wf.FatalError(err)
	}
	if err := rootCmd.Execute(); err != nil {
		wf.FatalError(err)
	}
}
