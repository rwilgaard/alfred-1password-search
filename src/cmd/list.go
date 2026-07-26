package cmd

import (
	"fmt"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
)

var (
	vaultFlag string

	listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list vault items",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		Run: func(_ *cobra.Command, args []string) {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			vault := cfg.VaultName
			if vaultFlag != "" {
				vault = vaultFlag
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

			var items []onepassword.Item
			for _, item := range rawItems {
				if vault == "" || item.VaultName == vault || item.VaultID == vault {
					if !cfg.FuzzySearch && len(query) > 0 {
						titleMatch := strings.Contains(strings.ToLower(item.Title), strings.ToLower(query))
						vaultMatch := strings.Contains(strings.ToLower(item.VaultName), strings.ToLower(query))
						urlMatch := strings.Contains(strings.ToLower(item.PrimaryURL), strings.ToLower(query))
						if !titleMatch && !vaultMatch && !urlMatch {
							continue
						}
					}
					items = append(items, item)
				}
			}

			updateAvailable := util.IsCacheStale(wf, itemsCacheKey) && (len(query) == 0 || strings.Contains("update items sync", strings.ToLower(query)))

			if updateAvailable {
				wf.Configure(aw.SuppressUIDs(true))
			}

			for _, item := range items {
				subtitle := item.VaultName
				if item.Category != "" {
					subtitle += "  •  " + item.Category
				}

				if item.PrimaryURL != "" {
					subtitle += "  •  " + item.PrimaryURL
				}

				it := wf.NewItem(item.Title).
					Subtitle(subtitle).
					Match(fmt.Sprintf("%s %s %s %s %s", item.ID, item.Title, item.VaultName, item.Category, item.PrimaryURL)).
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

				if cfg.ModifierCmd != "" {
					it.NewModifier(aw.ModCmd).
						Subtitle(cfg.ModifierCmd).
						Var("action", cfg.ModifierCmd).
						Arg(cfg.ModifierCmd).
						Valid(true)
				}
				if cfg.ModifierOpt != "" {
					it.NewModifier(aw.ModOpt).
						Subtitle(cfg.ModifierOpt).
						Var("action", cfg.ModifierOpt).
						Arg(cfg.ModifierOpt).
						Valid(true)
				}
				if cfg.ModifierCtrl != "" {
					it.NewModifier(aw.ModCtrl).
						Subtitle(cfg.ModifierCtrl).
						Var("action", cfg.ModifierCtrl).
						Arg(cfg.ModifierCtrl).
						Valid(true)
				}
			}

			if cfg.FuzzySearch && len(query) > 0 {
				wf.Filter(query)
			}

			if updateAvailable {
				wf.NewItem("Update items").
					Subtitle("1Password vault has changed. Press ⏎ to update.").
					Icon(util.IconFolder).
					Match("Update items sync").
					Var("item_id", "sync").
					Var("action", "Show Details").
					Var("list_query", query).
					Arg("sync").
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

func init() {
	listCmd.Flags().StringVarP(&vaultFlag, "vault", "v", "", "filter by vault name (overrides vault_name env var)")
	rootCmd.AddCommand(listCmd)
}
