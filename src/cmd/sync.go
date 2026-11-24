package cmd

import (
	"fmt"
	"log"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/spf13/cobra"
)

var reopenFlag bool

var syncCmd = &cobra.Command{
	Use:          "sync",
	Short:        "Sync items and vaults from 1Password",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		log.Println("Syncing 1Password items...")
		if err := syncCache(); err != nil {
			return reportError(fmt.Errorf("sync failed: %w", err))
		}
		log.Println("Sync complete!")
		if reopenFlag {
			// Show the refreshed list with the previous query restored.
			if err := wf.Alfred.RunTrigger("list_items", cfg.ListQuery); err != nil {
				log.Printf("reopen list: %v", err)
			}
		}
		return nil
	},
}

// syncCache fetches vaults and items from 1Password, writes them to the
// workflow cache, and records the 1Password DB mtime so the cache counts as fresh.
func syncCache() error {
	svc, err := onepassword.NewService("op")
	if err != nil {
		return err
	}

	vaults, err := svc.ListVaults()
	if err != nil {
		return fmt.Errorf("could not fetch vaults: %w", err)
	}
	if err := wf.Cache.StoreJSON(vaultsCacheKey, vaults); err != nil {
		return fmt.Errorf("could not write vault cache: %w", err)
	}

	items, err := svc.ListItems("")
	if err != nil {
		return fmt.Errorf("could not fetch items: %w", err)
	}
	if err := wf.Cache.StoreJSON(itemsCacheKey, items); err != nil {
		return fmt.Errorf("could not write item cache: %w", err)
	}

	util.SaveLastSyncDbTime(wf)
	return nil
}

func init() {
	syncCmd.Flags().BoolVar(&reopenFlag, "reopen", false, "reopen the item list after syncing")
	rootCmd.AddCommand(syncCmd)
}
