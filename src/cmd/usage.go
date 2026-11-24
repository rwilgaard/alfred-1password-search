package cmd

import (
	"log"
	"time"

	"github.com/rwilgaard/alfred-1password-search/src/internal/usage"
	"github.com/spf13/cobra"
)

// usageDataKey is stored in the data dir, not the cache, so clearing the cache keeps the learned ranking.
const usageDataKey = "usage.json"

func loadUsage() usage.Store {
	s := usage.Store{}
	if err := wf.Data.LoadJSONIfExists(usageDataKey, &s); err != nil {
		log.Printf("load usage: %v", err)
		return usage.Store{}
	}
	return s
}

func recordUse(id string) {
	s := loadUsage()
	s.Record(id, time.Now())
	if err := wf.Data.StoreJSON(usageDataKey, s); err != nil {
		log.Printf("store usage: %v", err)
	}
}

// recordCmd counts uses that don't otherwise run Go code, like Open URL.
var recordCmd = &cobra.Command{
	Use:    "record <item_id>",
	Short:  "count a use of an item",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		recordUse(args[0])
	},
}

func init() {
	rootCmd.AddCommand(recordCmd)
}
