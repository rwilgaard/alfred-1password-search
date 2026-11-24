package cmd

import (
	"fmt"
	"log"

	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
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

// syncCache fetches the items from 1Password, writes them to the workflow
// cache, and records the 1Password DB mtime so the cache counts as fresh.
func syncCache() error {
	items, err := onepassword.NewService().ListItems()
	if err != nil {
		return fmt.Errorf("could not fetch items: %w", err)
	}
	if err := wf.Cache.StoreJSON(itemsCacheKey, items); err != nil {
		return fmt.Errorf("could not write item cache: %w", err)
	}
	util.SaveLastSyncDbTime(wf)
	return nil
}

// cachedItems loads the item cache for a Script Filter. On first run there
// is none: it starts a sync, sends a placeholder that reruns until the data
// is there, and returns false.
func cachedItems() ([]onepassword.Item, bool) {
	if !wf.Cache.Exists(itemsCacheKey) {
		wf.Rerun(0.5)
		wf.NewItem("Fetching 1Password items...").
			Subtitle("Please wait a moment").
			Icon(aw.IconInfo)
		if err := alfredutils.RefreshCache(wf, itemsCacheKey, cacheMaxAge, []string{syncJobName}); err != nil {
			wf.FatalError(err)
		}
		alfredutils.HandleFeedback(wf)
		return nil, false
	}
	var items []onepassword.Item
	if err := wf.Cache.LoadJSON(itemsCacheKey, &items); err != nil {
		wf.FatalError(err)
	}
	return items, true
}

func init() {
	syncCmd.Flags().BoolVar(&reopenFlag, "reopen", false, "reopen the item list after syncing")
	rootCmd.AddCommand(syncCmd)
}
