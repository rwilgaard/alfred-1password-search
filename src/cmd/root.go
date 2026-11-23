package cmd

import (
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
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
	IntelligentOrdering   bool   `env:"intelligent_ordering"`
	VaultName             string `env:"vault_name"`
	SelectedVaults        string `env:"selected_vaults"`
	SelectedVaultsKeyword string `env:"selected_vaults_keyword"`
	ListQuery             string `env:"list_query"`
}

const repo = "rwilgaard/alfred-1password-search"

const (
	itemsCacheKey  = "items.json"
	vaultsCacheKey = "vaults.json"
	syncJobName    = "sync"
)

var cacheMaxAge = 120 * time.Minute

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
	if cfg.IntelligentOrdering {
		wf.Configure(aw.SuppressUIDs(false))
	}
	if err := rootCmd.Execute(); err != nil {
		wf.FatalError(err)
	}
}

func setupService() *onepassword.Service {
	svc, err := onepassword.NewService("op")
	if err != nil {
		wf.FatalError(err)
		return nil
	}
	return svc
}

func init() {
	sopts := []fuzzy.Option{
		fuzzy.AdjacencyBonus(10.0),
		fuzzy.LeadingLetterPenalty(-0.1),
		fuzzy.MaxLeadingLetterPenalty(-3.0),
		fuzzy.UnmatchedLetterPenalty(-0.5),
	}
	wf = aw.New(
		aw.SortOptions(sopts...),
		aw.MaxResults(25),
		update.GitHub(repo),
		aw.SuppressUIDs(true),
	)
}
