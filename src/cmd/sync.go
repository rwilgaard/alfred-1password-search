package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/v2/alfred"
	"github.com/spf13/cobra"
)

const (
	// syncLockedKey marks that an automatic sync was skipped because
	// 1Password was locked. The list then offers a manual update instead.
	syncLockedKey = "sync_locked"
	lockedRetry   = 5 * time.Minute
	// minSyncInterval limits automatic syncs, since the 1Password database
	// changes far more often than the item list does.
	minSyncInterval = time.Minute
	// syncingKey marks a sync run by syncAndReopen. It isn't a background
	// job, so wf.IsRunning can't see it. A marker older than syncingMaxAge is
	// left over from a crash and ignored.
	syncingKey    = "syncing"
	syncingMaxAge = 2 * time.Minute
)

var (
	reopenFlag bool
	autoFlag   bool
)

var syncCmd = &cobra.Command{
	Use:          "sync",
	Short:        "Sync items from 1Password",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		svc := onepassword.NewService()
		switch {
		case reopenFlag:
			return syncAndReopen(svc)
		case autoFlag && !svc.IsAuthorized():
			// Never prompt for Touch ID while the user is typing a search.
			log.Println("1Password is locked; skipping automatic sync")
			return wf.Cache.Store(syncLockedKey, []byte{})
		}
		if err := syncCache(svc); err != nil {
			return reportError(fmt.Errorf("sync failed: %w", err))
		}
		return nil
	},
}

// syncAndReopen syncs inside the Run Script Alfred started, because an
// unlock there carries over to later op calls and a detached job would
// prompt again.
func syncAndReopen(svc *onepassword.Service) error {
	if err := svc.Authorize(); err != nil {
		return reportError(err)
	}
	if err := wf.Cache.Store(syncingKey, []byte{}); err != nil {
		return reportError(err)
	}
	defer func() { _ = wf.Cache.Delete(syncingKey) }()
	if err := wf.RunTrigger("list_items", cfg.ListQuery); err != nil {
		log.Printf("reopen list: %v", err)
	}
	if err := syncCache(svc); err != nil {
		return reportError(fmt.Errorf("sync failed: %w", err))
	}
	return nil
}

func syncCache(svc *onepassword.Service) error {
	items, err := svc.ListItems()
	if err != nil {
		return fmt.Errorf("could not fetch items: %w", err)
	}
	if err := wf.Cache.StoreJSON(itemsCacheKey, items); err != nil {
		return fmt.Errorf("could not write item cache: %w", err)
	}
	util.SaveLastSyncDbTime(wf.Cache)
	return wf.Cache.Delete(syncLockedKey)
}

// cachedItems returns false after sending a placeholder when there is no
// cache yet. The caller must then return.
func cachedItems() ([]onepassword.Item, bool) {
	if !wf.Cache.Exists(itemsCacheKey) {
		if _, err := wf.Start(syncJobName, "sync"); err != nil {
			wf.FatalError(err)
		}
		wf.Rerun(500 * time.Millisecond)
		wf.NewItem("Fetching 1Password items...").
			Subtitle("Please wait a moment").
			Icon(alfred.IconInfo)
		wf.Send()
		return nil, false
	}
	var items []onepassword.Item
	if err := wf.Cache.LoadJSON(itemsCacheKey, &items); err != nil {
		wf.FatalError(err)
	}
	return items, true
}

type syncState int

const (
	syncFresh syncState = iota
	syncRunning
	syncLocked // stale, and only a manual update can unlock
)

// checkSync also starts an automatic sync when the cache is stale.
func checkSync() syncState {
	if wf.IsRunning(syncJobName) || !wf.Cache.Expired(syncingKey, syncingMaxAge) {
		return syncRunning
	}
	if !util.IsCacheStale(wf.Cache, itemsCacheKey) || !wf.Cache.Expired(itemsCacheKey, minSyncInterval) {
		return syncFresh
	}
	if !wf.Cache.Expired(syncLockedKey, lockedRetry) {
		return syncLocked
	}
	if _, err := wf.Start(syncJobName, "sync", "--auto"); err != nil {
		log.Printf("start sync: %v", err)
		return syncFresh
	}
	return syncRunning
}

func init() {
	syncCmd.Flags().BoolVar(&reopenFlag, "reopen", false, "unlock, sync in the background, and reopen the item list")
	syncCmd.Flags().BoolVar(&autoFlag, "auto", false, "skip the sync instead of prompting if 1Password is locked")
	rootCmd.AddCommand(syncCmd)
}
