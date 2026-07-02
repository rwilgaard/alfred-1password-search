package cmd

import (
	"fmt"
	"log"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:          "sync",
	Short:        "Sync items and vaults from 1Password",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		log.Println("Syncing 1Password items...")

		svc, err := onepassword.NewService("op")
		if err != nil {
			msg := fmt.Sprintf("Sync failed: %s", err)
			if zerr := wf.Alfred.RunTrigger("error", msg); zerr != nil {
				log.Printf("Alfred error trigger failed: %v", zerr)
			}
			return err
		}

		vaults, err := svc.ListVaults()
		if err != nil {
			msg := fmt.Sprintf("Sync failed: could not fetch vaults: %s", err)
			if zerr := wf.Alfred.RunTrigger("error", msg); zerr != nil {
				log.Printf("Alfred error trigger failed: %v", zerr)
			}
			return err
		}
		if err := wf.Cache.StoreJSON(vaultsCacheKey, vaults); err != nil {
			msg := fmt.Sprintf("Sync failed: could not write vault cache: %s", err)
			if zerr := wf.Alfred.RunTrigger("error", msg); zerr != nil {
				log.Printf("Alfred error trigger failed: %v", zerr)
			}
			return err
		}

		items, err := svc.ListItems("")
		if err != nil {
			msg := fmt.Sprintf("Sync failed: could not fetch items: %s", err)
			if zerr := wf.Alfred.RunTrigger("error", msg); zerr != nil {
				log.Printf("Alfred error trigger failed: %v", zerr)
			}
			return err
		}
		if err := wf.Cache.StoreJSON(itemsCacheKey, items); err != nil {
			msg := fmt.Sprintf("Sync failed: could not write item cache: %s", err)
			if zerr := wf.Alfred.RunTrigger("error", msg); zerr != nil {
				log.Printf("Alfred error trigger failed: %v", zerr)
			}
			return err
		}

		log.Println("Sync complete!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
