// src/cmd/show.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:          "show",
	Short:        "show item details",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		itemID := args[0]

		if itemID == "sync" {
			if err := syncCache(); err != nil {
				return err
			}

			// Reopen Alfred showing the list of items with the previous query restored
			_ = wf.Alfred.RunTrigger("list_items", cfg.ListQuery)

			return nil
		}

		svc := setupService()
		if svc == nil {
			return nil
		}

		item, err := svc.GetItem(itemID)
		if err != nil {
			return err
		}

		var accountID string
		accountIDCacheKey := "account_id.json"
		if wf.Cache.Exists(accountIDCacheKey) {
			_ = wf.Cache.LoadJSON(accountIDCacheKey, &accountID)
		}

		if accountID == "" {
			var err error
			accountID, err = svc.GetAccountID()
			if err != nil {
				return err
			}
			_ = wf.Cache.StoreJSON(accountIDCacheKey, accountID)
		}

		wf.NewItem("← Go back").
			Icon(util.IconBack).
			Arg("go_back").
			Var("list_query", cfg.ListQuery).
			Valid(true)

		wf.NewItem("Name").
			Icon(util.GetIcon("name")).
			Subtitle(fmt.Sprintf("%s/%s", item.VaultName, item.Title)).
			Arg(item.Title).
			Var("sensitive", "false").
			Valid(true)

		for _, f := range item.Fields {
			if f.Value == "" {
				continue
			}
			if f.Type == "OTP" {
				continue
			}
			if f.Purpose == "NOTES" {
				wf.NewItem("Notes").
					Icon(util.GetIcon("notes")).
					Subtitle("Press ⏎ to show notes").
					Arg("notes").
					Var("notes_value", f.Value).
					Valid(true)
				continue
			}

			sub := f.Value
			sensitive := "false"
			if f.Type == "CONCEALED" {
				sub = strings.Repeat("•", 32)
				sensitive = "true"
			}

			wf.NewItem(f.Label).
				Icon(util.GetIcon(strings.ToLower(f.Label))).
				Subtitle(sub).
				Arg(f.Value).
				Var("sensitive", sensitive).
				Var("field_label", f.Label).
				Valid(true)
		}

		wf.NewItem("Open in 1Password").
			Icon(util.IconEdit).
			Arg("open").
			Var("account_id", accountID).
			Var("vault_id", item.VaultID).
			Var("item_id", item.ID).
			Valid(true)

		alfredutils.HandleFeedback(wf)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
