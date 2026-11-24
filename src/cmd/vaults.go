package cmd

import (
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/spf13/cobra"
)

var vaultsCmd = &cobra.Command{
	Use:          "vaults",
	Short:        "list vaults",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		items, ok := cachedItems()
		if !ok {
			return
		}

		wf.NewItem("All Vaults").
			Subtitle("Show items from all vaults").
			Icon(util.IconFolder).
			Match("*").
			Var("vault_name", "").
			Valid(true)

		for _, v := range onepassword.Vaults(items) {
			wf.NewItem(v.Name).
				UID(v.ID).
				Icon(util.IconFolder).
				Var("vault_name", v.Name).
				Valid(true)
		}

		wf.Filter(args[0])
		wf.Send()
	},
}

func init() {
	rootCmd.AddCommand(vaultsCmd)
}
