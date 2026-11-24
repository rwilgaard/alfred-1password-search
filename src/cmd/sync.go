package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/spf13/cobra"
)

const (
	// syncLockedKey marks that an automatic sync was skipped because
	// 1Password was locked. The list then offers a manual update instead.
	syncLockedKey = "sync_locked"
	// lockedRetry is how long the list waits before trying an automatic sync
	// again after one found 1Password locked.
	lockedRetry = 5 * time.Minute
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

// syncAndReopen backs the list's "Update items" row. It runs in the Run
// Script Alfred started, where an unlock carries over to later op calls (a
// detached job would prompt again). It reopens the list first, which shows
// "Updating items…" while the syncingKey marker exists, then syncs.
func syncAndReopen(svc *onepassword.Service) error {
	if err := svc.Authorize(); err != nil {
		return reportError(err)
	}
	if err := wf.Cache.Store(syncingKey, []byte{}); err != nil {
		return reportError(err)
	}
	defer func() { _ = wf.Cache.Store(syncingKey, nil) }() // nil deletes the marker
	if err := wf.Alfred.RunTrigger("list_items", cfg.ListQuery); err != nil {
		log.Printf("reopen list: %v", err)
	}
	if err := syncCache(svc); err != nil {
		return reportError(fmt.Errorf("sync failed: %w", err))
	}
	return nil
}

// syncCache fetches the items from 1Password, writes them to the workflow
// cache, and records the 1Password DB mtime so the cache counts as fresh.
func syncCache(svc *onepassword.Service) error {
	items, err := svc.ListItems()
	if err != nil {
		return fmt.Errorf("could not fetch items: %w", err)
	}
	if err := wf.Cache.StoreJSON(itemsCacheKey, items); err != nil {
		return fmt.Errorf("could not write item cache: %w", err)
	}
	util.SaveLastSyncDbTime(wf)
	return wf.Cache.Store(syncLockedKey, nil) // nil deletes the marker
}

// startJob runs this binary with args as a detached background job. A job
// of the same name that is already running counts as started.
func startJob(name string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	err = wf.RunInBackground(name, exec.Command(exe, args...))
	if errors.Is(err, aw.ErrJobExists{}) {
		return nil
	}
	return err
}

// cachedItems loads the item cache for a Script Filter. On first run there
// is none: it starts a sync, sends a placeholder that reruns until the data
// is there, and returns false.
func cachedItems() ([]onepassword.Item, bool) {
	if !wf.Cache.Exists(itemsCacheKey) {
		if err := startJob(syncJobName, "sync"); err != nil {
			wf.FatalError(err)
		}
		wf.Rerun(0.5)
		wf.NewItem("Fetching 1Password items...").
			Subtitle("Please wait a moment").
			Icon(aw.IconInfo)
		wf.SendFeedback()
		return nil, false
	}
	var items []onepassword.Item
	if err := wf.Cache.LoadJSON(itemsCacheKey, &items); err != nil {
		wf.FatalError(err)
	}
	return items, true
}

// syncState is what the list should show about the cache.
type syncState int

const (
	syncFresh   syncState = iota
	syncRunning           // a sync is in progress
	syncLocked            // stale, and 1Password must be unlocked to update
)

// checkSync starts an automatic background sync when the cache is stale and
// reports what the list should show.
func checkSync() syncState {
	if wf.IsRunning(syncJobName) || !wf.Cache.Expired(syncingKey, syncingMaxAge) {
		return syncRunning
	}
	if !util.IsCacheStale(wf, itemsCacheKey) || !wf.Cache.Expired(itemsCacheKey, minSyncInterval) {
		return syncFresh
	}
	if !wf.Cache.Expired(syncLockedKey, lockedRetry) {
		return syncLocked
	}
	if err := startJob(syncJobName, "sync", "--auto"); err != nil {
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
