package cmd

import (
	"fmt"
	"log"

	"github.com/rwilgaard/alfred-1password-search/src/internal/clipboard"
	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/totp"
	"github.com/spf13/cobra"
)

// copyFields maps the list's copy actions (the modifier_* choices) to fields.
var copyFields = map[string]string{
	"Copy Password": onepassword.KeyPassword,
	"Copy Username": onepassword.KeyUsername,
	"Copy OTP":      onepassword.KeyOTP,
}

var copyCmd = &cobra.Command{
	Use:   "copy <item_id> <action>",
	Short: "copy a field of an item to the clipboard",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		key, ok := copyFields[args[1]]
		if !ok {
			return reportError(fmt.Errorf("unknown copy action %q", args[1]))
		}
		msg, err := copyField(args[0], key)
		if err != nil {
			return reportError(err)
		}
		fmt.Println(msg) // shown by the notification after the Run Script
		return nil
	},
}

func copyField(itemID, key string) (string, error) {
	svc := onepassword.NewService()
	item, err := svc.GetItem(itemID)
	if err != nil {
		return "", err
	}
	recordUse(item.ID)

	f, ok := item.Lookup(key)
	if !ok {
		return "", fmt.Errorf("%s has no %s", item.Title, key)
	}
	value := f.Value
	if key == onepassword.KeyOTP {
		if value, err = totp.Current(f.Value); err != nil {
			return "", err
		}
	}
	return copySecret(value, f.Label)
}

// copySecret puts value on the clipboard and returns a message for the user.
func copySecret(value, label string) (string, error) {
	if err := clipboard.Copy(value); err != nil {
		return "", err
	}
	return "Copied " + label, nil
}

// reportError shows err through the workflow's "error" trigger. Run Script
// and background commands can't use wf.FatalError: nothing reads their output.
func reportError(err error) error {
	if terr := wf.Alfred.RunTrigger("error", err.Error()); terr != nil {
		log.Printf("Alfred error trigger failed: %v", terr)
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
