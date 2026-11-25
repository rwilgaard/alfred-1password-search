package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/rwilgaard/go-alfredutils/v2/alfred/totp"
	alfredui "github.com/rwilgaard/go-alfredutils/v2/alfred/ui"
	"github.com/spf13/cobra"
)

// Variables used by workflow/details.alfredui. Secrets live only in the UI's
// memory while the window is open and reach scripts via the environment.
const (
	envTOTPSecret = "totp_secret"
	envNotes      = "notes"
	envCopyKey    = "copy_key"
	envCopyValue  = "copy_value"
	envCopyLabel  = "copy_label"
	envReopened   = "reopened" // set by the show_details trigger
)

// Component ids in workflow/details.alfredui/ui.json.
const (
	uiTitle    = "title"
	uiSubtitle = "subtitle"
	uiTOTPBox  = "totpbox"
	uiTOTPCode = "totpcode"
	uiTOTPBar  = "totpbar"
	uiFields   = "fields"
	uiOpenURL  = "openurl"
	uiOpen1P   = "open1p"
	uiNotesBtn = "notesbtn"
	uiLoading  = "loading"
	uiStatus   = "status"
)

const (
	// uiScript runs the binary from a UI script action, relative to the .alfredui folder.
	uiScript           = "../scripts/run.sh"
	accountIDCacheKey  = "account_id.json"
	totpWarnSeconds    = 5
	showDetailsTrigger = "show_details"
	// unlockPromptThreshold separates a normal op item get from one that
	// waited on 1Password's unlock prompt.
	unlockPromptThreshold = 1500 * time.Millisecond
)

// The ui-* commands are run by Alfred UI script actions. Their stdout must be
// JSON actions only, so they use Run (a returned error would make run() call
// wf.FatalError) and never send Script Filter feedback.

var uiDetailsCmd = &cobra.Command{
	Use:    "ui-details",
	Short:  "fill in the item details window",
	Hidden: true,
	Run: func(_ *cobra.Command, _ []string) {
		emit(uiDetails(cfg.ItemID))
	},
}

var uiTOTPCmd = &cobra.Command{
	Use:    "ui-totp",
	Short:  "refresh the TOTP code (pulse)",
	Hidden: true,
	Run: func(_ *cobra.Command, _ []string) {
		emit(uiTOTP(os.Getenv(envTOTPSecret), time.Now()))
	},
}

var uiNotesCmd = &cobra.Command{
	Use:    "ui-notes",
	Short:  "show the notes sheet",
	Hidden: true,
	Run: func(_ *cobra.Command, _ []string) {
		emit(alfredui.Actions{Sheet: notesSheet(os.Getenv(envNotes))})
	},
}

var uiCopyCmd = &cobra.Command{
	Use:       "ui-copy close|stay",
	Short:     "copy a value from the details window",
	Hidden:    true,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"close", "stay"},
	Run: func(_ *cobra.Command, args []string) {
		emit(uiCopy(os.Getenv(envCopyKey), os.Getenv(envCopyValue), os.Getenv(envCopyLabel), args[0] == "close"))
	},
}

func uiDetails(itemID string) alfredui.Actions {
	fail := func(err error) alfredui.Actions {
		log.Printf("[ui] %v", err)
		return alfredui.NewUpdate().Status(uiLoading, err.Error()+"\nPress Esc to go back.", alfredui.ColourError)
	}
	if itemID == "" {
		return fail(errors.New("no item selected"))
	}
	svc := onepassword.NewService()
	start := time.Now()
	item, err := svc.GetItem(itemID)
	if err != nil {
		return fail(err)
	}
	// A slow fetch means 1Password asked to unlock. The prompt took focus,
	// which closed Alfred and this window, so open the window again. The
	// reopened window never does this, so it can't loop.
	if time.Since(start) > unlockPromptThreshold && os.Getenv(envReopened) == "" {
		if err := wf.RunTrigger(showDetailsTrigger, itemID); err != nil {
			log.Printf("[ui] reopen details: %v", err)
		} else {
			return alfredui.Actions{}
		}
	}
	recordUse(item.ID)

	subtitle := item.VaultName
	if item.Category != "" {
		subtitle += "  •  " + item.Category
	}
	a := alfredui.NewUpdate().
		Set(uiTitle, alfredui.Props{"text": item.Title}).
		Set(uiSubtitle, alfredui.Props{"text": subtitle}).
		Set(uiFields, alfredui.Props{"data": fieldsData(util.DetailRows(item))}).
		Set(uiLoading, alfredui.Props{"hidden": true})

	a = a.SetVar("item_url", item.PrimaryURL).Enable(item.PrimaryURL != "", uiOpenURL)

	// The details still work without the account id; only "Open in 1Password" needs it.
	accountID, err := cachedAccountID(svc)
	if err != nil {
		log.Printf("[ui] account id: %v", err)
	}
	a = a.SetVar("op_url", fmt.Sprintf("onepassword://view-item/?a=%s&v=%s&i=%s", accountID, item.VaultID, item.ID)).
		Enable(accountID != "", uiOpen1P)

	notes := item.Notes()
	a = a.SetVar(envNotes, notes).Enable(notes != "", uiNotesBtn)
	if item.Category == "SECURE_NOTE" && notes != "" {
		a = a.WithSheet(notesSheet(notes))
	}

	if f, ok := item.Lookup(onepassword.KeyOTP); ok && f.Value != "" {
		a = a.SetVar(envTOTPSecret, f.Value).Set(uiTOTPBox, alfredui.Props{"hidden": false})
		for id, p := range uiTOTP(f.Value, time.Now()).Update.IDs {
			a = a.Set(id, p)
		}
		a.Pulse = &alfredui.Pulse{Interval: 1}
		// The code label was laid out as " " and keeps that width until a relayout.
		a.Update.Relayout = true
	}
	return a
}

// fieldsData is the dataview's data provider: the rows plus a filter on the
// "filter" text field. Set as a whole because an update replaces it.
func fieldsData(rows []map[string]string) map[string]any {
	return map[string]any{
		"type": "basic",
		"src":  rows,
		"filter": map[string]any{
			"type":  "basic",
			"args":  []string{"name.text", "value.text"},
			"match": "{value:filter.text}",
		},
	}
}

func uiTOTP(secret string, now time.Time) alfredui.Actions {
	k, err := totp.Parse(secret)
	if err != nil {
		return alfredui.NewUpdate().Status(uiStatus, fmt.Sprintf("Can't generate the one-time password: %s", err), alfredui.ColourError)
	}
	code, remaining := k.Code(now)
	colour := alfredui.ColourInfo
	if remaining <= totpWarnSeconds {
		colour = alfredui.ColourError
	}
	return alfredui.NewUpdate().
		Set(uiTOTPCode, alfredui.Props{"text": totp.Group(code)}).
		Set(uiTOTPBar, alfredui.Props{"progress": float64(remaining) / float64(k.Period), "colour": colour})
}

func uiCopy(key, value, label string, closeAfter bool) alfredui.Actions {
	if key == onepassword.KeyOTP {
		var err error
		if value, err = totp.Current(os.Getenv(envTOTPSecret)); err != nil {
			return alfredui.NewUpdate().Status(uiStatus, err.Error(), alfredui.ColourError)
		}
		label = "one-time password"
	}
	if value == "" {
		return alfredui.NewUpdate().Status(uiStatus, "Nothing to copy.", alfredui.ColourError)
	}
	msg, err := copySecret(value, label)
	if err != nil {
		return alfredui.NewUpdate().Status(uiStatus, err.Error(), alfredui.ColourError)
	}
	if closeAfter {
		return alfredui.Actions{Finish: &alfredui.Finish{Args: msg}}
	}
	return alfredui.NewUpdate().Status(uiStatus, msg, alfredui.ColourSuccess)
}

func notesSheet(notes string) *alfredui.Sheet {
	copyNotes := []map[string]any{{
		"triggers": "actioned",
		"action": map[string]any{
			"type": "script", "src": uiScript, "args": []string{"ui-copy", "close"}, "async": true,
			"variables": map[string]string{envCopyKey: "notes", envCopyValue: "{var:notes}", envCopyLabel: "notes"},
		},
	}}
	return &alfredui.Sheet{
		ID: "notessheet",
		Root: map[string]any{
			"type": "view", "layout": "border", "spacing": 8, "padding": 16,
			"components": []map[string]any{
				{"constraints": map[string]string{"region": "top"}, "type": "label", "text": "Notes", "font": map[string]string{"size": "large", "weight": "bold"}},
				{
					"constraints": map[string]any{"region": "centre", "size": map[string]int{"width": 480, "height": 320}},
					"type":        "markdownarea", "text": notes, "behaviour": map[string]string{"scroll": "start"},
				},
				{"constraints": map[string]string{"region": "bottom"}, "type": "button", "text": "Copy Notes", "shortcut": "⌘C", "events": copyNotes},
			},
		},
	}
}

func cachedAccountID(svc *onepassword.Service) (string, error) {
	var id string
	if err := wf.Cache.LoadJSONIfExists(accountIDCacheKey, &id); err == nil && id != "" {
		return id, nil
	}
	id, err := svc.GetAccountID()
	if err != nil {
		return "", err
	}
	if err := wf.Cache.StoreJSON(accountIDCacheKey, id); err != nil {
		log.Printf("[ui] cache account id: %v", err)
	}
	return id, nil
}

func emit(a alfredui.Actions) {
	if err := alfredui.Emit(a); err != nil {
		log.Printf("[ui] emit failed: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(uiDetailsCmd, uiTOTPCmd, uiNotesCmd, uiCopyCmd)
}
