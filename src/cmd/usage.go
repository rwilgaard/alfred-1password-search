package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// recordUse counts a use of an item for ranking and the recent row.
func recordUse(id string) {
	if err := wf.Usage.Record(id); err != nil {
		log.Printf("record usage: %v", err)
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
