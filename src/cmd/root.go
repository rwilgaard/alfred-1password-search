package cmd

import (
	"os"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
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
	wf  *aw.Workflow
	cfg = &workflowConfig{}

	rootCmd = &cobra.Command{
		Use:           "alfred-1password-search",
		Short:         "Alfred workflow for searching 1Password vault items",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func Execute() {
	wf.Run(run)
}

func run() {
	if err := alfredutils.InitWorkflow(wf, cfg); err != nil {
		wf.FatalError(err)
	}
	if err := alfredutils.CheckForUpdates(wf); err != nil {
		wf.FatalError(err)
	}
	if err := rootCmd.Execute(); err != nil {
		wf.FatalError(err)
	}
}

func init() {
	sopts := []fuzzy.Option{
		fuzzy.AdjacencyBonus(10.0),
		fuzzy.LeadingLetterPenalty(-0.1),
		fuzzy.MaxLeadingLetterPenalty(-3.0),
		fuzzy.UnmatchedLetterPenalty(-0.5),
	}
	opts := []aw.Option{
		aw.SortOptions(sopts...),
		aw.MaxResults(25),
	}
	// Alfred UI scripts get no alfred_workflow_version, and the updater panics without it.
	if os.Getenv("alfred_workflow_version") != "" {
		opts = append(opts, update.GitHub(repo))
	}
	wf = aw.New(opts...)
}
