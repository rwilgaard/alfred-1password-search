package cmd

import (
	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
	"go.deanishe.net/fuzzy"
)

type workflowConfig struct {
}

const (
    repo            = "rwilgaard/alfred-template"
    keychainAccount = "alfred-template"
)

var (
    wf      *aw.Workflow
    cfg     = &workflowConfig{}
    rootCmd = &cobra.Command{
        Use:   "alfred-template",
        Short: "alfred-template is a CLI to be used by Alfred for ...",
    }
)

func Execute() {
    wf.Run(run)
}

func run() {
    alfredutils.AddClearAuthMagic(wf, keychainAccount)

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
    wf = aw.New(
        aw.SortOptions(sopts...),
        update.GitHub(repo),
    )
}
