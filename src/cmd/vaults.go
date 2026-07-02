package cmd

import (
	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
)

var vaultsCmd = &cobra.Command{
	Use:          "vaults",
	Short:        "list vaults",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var vaults []onepassword.Vault
		if err := alfredutils.LoadCache(wf, vaultsCacheKey, &vaults); err != nil {
			wf.FatalError(err)
		}

		if !wf.Cache.Exists(vaultsCacheKey) {
			wf.Rerun(0.5)
			wf.NewItem("Fetching 1Password vaults...").
				Subtitle("Please wait a moment").
				Icon(aw.IconInfo)
			// If cache doesn't exist, we must run RefreshCache regardless of lock status to initialize the workflow.
			if err := alfredutils.RefreshCache(wf, vaultsCacheKey, cacheMaxAge, []string{syncJobName}); err != nil {
				wf.FatalError(err)
			}
			alfredutils.HandleFeedback(wf)
			return
		}

		wf.NewItem("All Vaults").
			Subtitle("Show items from all vaults").
			Icon(util.IconFolder).
			Match("*").
			Var("vault_name", "").
			Valid(true)

		for _, v := range vaults {
			wf.NewItem(v.Name).
				UID(v.ID).
				Icon(util.IconFolder).
				Var("vault_name", v.Name).
				Valid(true)
		}

		wf.Filter(args[0])
		alfredutils.HandleFeedback(wf)
	},
}

func init() {
	rootCmd.AddCommand(vaultsCmd)
}
