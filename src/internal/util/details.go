package util

import (
	"strings"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
)

// Mask is a fixed-width mask, so the secret's length isn't revealed.
var Mask = strings.Repeat("•", 16)

// DetailRows returns the rows for the details window's field list. OTP and
// notes have their own controls. Raw values sit in hidden cells for copy and reveal.
func DetailRows(item *onepassword.ItemDetail) []map[string]string {
	var rows []map[string]string
	add := func(key, label, display, raw string) {
		rows = append(rows, map[string]string{
			"name.text":  label,
			"value.text": firstLine(display),
			"raw.text":   raw,
			"key.text":   key,
		})
	}

	for _, f := range item.Fields {
		if f.Value == "" || f.Type == "OTP" || f.Purpose == "NOTES" {
			continue
		}
		display := f.Value
		if f.Type == "CONCEALED" {
			display = Mask
		}
		add(f.ID, f.Label, display, f.Value)
	}
	if item.PrimaryURL != "" {
		add("website", "website", item.PrimaryURL, item.PrimaryURL)
	}
	return rows
}

func firstLine(s string) string {
	if first, _, more := strings.Cut(s, "\n"); more {
		return first + " …"
	}
	return s
}
