package cmd

import (
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/usage"
	"github.com/rwilgaard/go-alfredutils/alfredutils"
	"github.com/spf13/cobra"
)

// recentTTL is how long the last used item stays pinned and reachable by the hotkey.
const recentTTL = 5 * time.Minute

// recentID returns the last used item if it was used within recentTTL.
func recentID(s usage.Store) (string, bool) {
	id, usedAt, ok := s.Latest()
	return id, ok && time.Since(usedAt) < recentTTL
}

// recentCmd backs the recent hotkey: it outputs the variables that route the
// workflow to the recent item's details.
var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "open the most recently used item",
	RunE: func(_ *cobra.Command, _ []string) error {
		av := aw.NewArgVars()
		id, ok := recentID(loadUsage())
		if !ok {
			return av.Arg("No recently used 1Password item").Var("action", "none").Send()
		}
		return av.Var("action", "show").Var("item_id", id).Var("item_name", cachedTitle(id)).Send()
	},
}

// cachedTitle looks up an item's title so the details window can show it before loading.
func cachedTitle(id string) string {
	var items []onepassword.Item
	if err := alfredutils.LoadCache(wf, itemsCacheKey, &items); err == nil {
		for _, it := range items {
			if it.ID == id {
				return it.Title
			}
		}
	}
	return " "
}

func init() {
	rootCmd.AddCommand(recentCmd)
}
