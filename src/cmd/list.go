package cmd

import (
	"slices"
	"strings"
	"time"

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

			items, ok := cachedItems()
			if !ok {
				return
			}
			if len(targetVaults) > 0 {
				items = slices.DeleteFunc(items, func(it onepassword.Item) bool {
					return !util.MatchesVault(it.VaultName, it.VaultID, targetVaults)
				})
			}

			usage := loadUsage()
			items = util.FilterAndRankItems(items, query, cfg.FuzzySearch, usage.Scores(time.Now()))

			// Sync status only shows on an empty query, or one asking for it, so
			// it doesn't push matches down while typing.
			showSync := query == "" || strings.Contains("update items sync", strings.ToLower(query))
			switch checkSync() {
			case syncRunning:
				wf.Rerun(0.5) // pick up the new items when the sync is done
				if showSync {
					wf.NewItem("Updating items…").
						Subtitle("Showing cached items until the update finishes").
						Icon(util.IconUpdate).
						Valid(false)
				}
			case syncLocked:
				if showSync {
					wf.NewItem("Update items").
						Subtitle("1Password is locked. Press ⏎ to unlock and update.").
						Icon(util.IconUpdate).
						Var("action", actionSync).
						Var("list_query", query).
						Arg(actionSync).
						Valid(true)
				}
			}
			if id, ok := recentID(usage); ok && query == "" {
				addRecentItem(items, id)
			}
			for _, item := range items {
				addListItem(item, query)
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

	// No UID: results are ranked here (match + frecency), so Alfred's knowledge must not reorder them.
	it := wf.NewItem(item.Title).
		Subtitle(subtitle).
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

// addRecentItem puts the recently used item first, with ⏎ opening its details.
func addRecentItem(items []onepassword.Item, id string) {
	for _, item := range items {
		if item.ID == id {
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
