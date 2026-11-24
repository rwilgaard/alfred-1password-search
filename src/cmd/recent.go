package cmd

import (
	"log"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/spf13/cobra"
)

const (
	recentCacheKey = "recent.json"
	recentTTL      = 5 * time.Minute
)

// recentItem is the last item copied from or opened, so the search and the
// recent hotkey can get back to it. No secrets.
type recentItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	UsedAt int64  `json:"used_at"`
}

func rememberItem(id, title string) {
	r := recentItem{ID: id, Title: title, UsedAt: time.Now().Unix()}
	if err := wf.Cache.StoreJSON(recentCacheKey, r); err != nil {
		log.Printf("remember recent item: %v", err)
	}
}

// loadRecent returns the recent item if it was used within recentTTL.
func loadRecent() (recentItem, bool) {
	var r recentItem
	if !wf.Cache.Exists(recentCacheKey) {
		return r, false
	}
	if err := wf.Cache.LoadJSON(recentCacheKey, &r); err != nil || r.ID == "" {
		return r, false
	}
	return r, time.Since(time.Unix(r.UsedAt, 0)) < recentTTL
}

// recentCmd backs the recent hotkey: it outputs the variables that route the
// workflow to the recent item's details.
var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "open the most recently used item",
	RunE: func(_ *cobra.Command, _ []string) error {
		av := aw.NewArgVars()
		r, ok := loadRecent()
		if !ok {
			return av.Arg("No recently used 1Password item").Var("action", "none").Send()
		}
		return av.Var("action", "show").Var("item_id", r.ID).Var("item_name", r.Title).Send()
	},
}

func init() {
	rootCmd.AddCommand(recentCmd)
}
