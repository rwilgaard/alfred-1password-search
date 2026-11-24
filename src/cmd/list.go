package cmd

import (
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
)

// action values routed on by the conditional after the list Script Filters.
const (
	actionShowDetails = "Show Details"
	actionSync        = "Sync"
)

var (
	vaultFlag  string
	vaultsFlag []string

	listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list vault items",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			var targetVaults []string
			if cmd.Flags().Changed("vaults") {
				targetVaults = util.ParseVaults(vaultsFlag)
				if len(targetVaults) == 0 {
					wf.NewItem("No vaults configured").
						Subtitle("Please configure Selected Vaults in workflow configuration").
						Icon(util.IconFolder).
						Valid(false)
					alfredutils.HandleFeedback(wf)
					return
				}
			} else if vaultFlag != "" {
				targetVaults = util.ParseVaults([]string{vaultFlag})
			} else if cfg.VaultName != "" {
				targetVaults = util.ParseVaults([]string{cfg.VaultName})
			}

			var rawItems []onepassword.Item
			if err := alfredutils.LoadCache(wf, itemsCacheKey, &rawItems); err != nil {
				wf.FatalError(err)
			}

			if !wf.Cache.Exists(itemsCacheKey) {
				wf.Rerun(0.5)
				wf.NewItem("Fetching 1Password items...").
					Subtitle("Please wait a moment").
					Icon(aw.IconInfo)
				// If cache doesn't exist, we must run RefreshCache regardless of lock status to initialize the workflow.
				if err := alfredutils.RefreshCache(wf, itemsCacheKey, cacheMaxAge, []string{syncJobName}); err != nil {
					wf.FatalError(err)
				}
				alfredutils.HandleFeedback(wf)
				return
			}

			var vaultFiltered []onepassword.Item
			for _, item := range rawItems {
				if util.MatchesVault(item.VaultName, item.VaultID, targetVaults) {
					vaultFiltered = append(vaultFiltered, item)
				}
			}

			items := util.FilterAndRankItems(vaultFiltered, query, cfg.FuzzySearch)

			updateAvailable := util.IsCacheStale(wf, itemsCacheKey) && (len(query) == 0 || strings.Contains("update items sync", strings.ToLower(query)))

			if updateAvailable {
				wf.Configure(aw.SuppressUIDs(true))
			}
			if query == "" {
				addRecentItem(items)
			}
			for _, item := range items {
				addListItem(item, query)
			}

			if updateAvailable {
				wf.NewItem("Update items").
					Subtitle("1Password vault has changed. Press ⏎ to update.").
					Icon(util.IconFolder).
					Match("Update items sync").
					Var("action", actionSync).
					Var("list_query", query).
					Arg(actionSync).
					Valid(true)

				items := wf.Feedback.Items
				if len(items) > 1 {
					item := items[len(items)-1]
					copy(items[1:], items[:len(items)-1])
					items[0] = item
				}
			}

			alfredutils.HandleFeedback(wf)
		},
	}
)

func addListItem(item onepassword.Item, query string) *aw.Item {
	subtitle := item.VaultName
	if item.Category != "" {
		subtitle += "  •  " + item.Category
	}
	if item.PrimaryURL != "" {
		subtitle += "  •  " + item.PrimaryURL
	}

	it := wf.NewItem(item.Title).
		Subtitle(subtitle).
		UID(item.ID).
		Var("item_id", item.ID).
		Var("item_name", item.Title).
		Var("item_url", item.PrimaryURL).
		Var("vault_id", item.VaultID).
		Var("vault_name", item.VaultName).
		Var("list_query", query).
		Var("action", cfg.ModifierReturn).
		Arg(cfg.ModifierReturn).
		Valid(true)

	for _, m := range []struct {
		key, action string
	}{
		{aw.ModCmd, cfg.ModifierCmd},
		{aw.ModOpt, cfg.ModifierOpt},
		{aw.ModCtrl, cfg.ModifierCtrl},
		{aw.ModShift, cfg.ModifierShift},
	} {
		if m.action != "" {
			it.NewModifier(m.key).
				Subtitle(m.action).
				Var("action", m.action).
				Arg(m.action).
				Valid(true)
		}
	}
	return it
}

// addRecentItem puts the recently used item first, with ⏎ opening its
// details. Alfred orders results by what it has learned for each UID, which
// would push it down, so this list is sent without UIDs. Typed searches keep
// them, so learned ranking still applies there.
func addRecentItem(items []onepassword.Item) {
	r, ok := loadRecent()
	if !ok {
		return
	}
	for _, item := range items {
		if item.ID == r.ID {
			wf.Configure(aw.SuppressUIDs(true)) // before any item is created
			addListItem(item, "").
				Title("↺ "+item.Title).
				Subtitle("Recently used  •  ⏎ to show details").
				Var("action", actionShowDetails).
				Arg(actionShowDetails)
			return
		}
	}
}

func init() {
	listCmd.Flags().StringVarP(&vaultFlag, "vault", "v", "", "filter by vault name (overrides vault_name env var)")
	listCmd.Flags().StringSliceVarP(&vaultsFlag, "vaults", "V", []string{}, "filter by vault names (comma-separated or multiple)")
	rootCmd.AddCommand(listCmd)
}
