package util

import (
	"reflect"
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
)

func TestDetailRows(t *testing.T) {
	item := &onepassword.ItemDetail{
		Title:      "Gmail",
		PrimaryURL: "https://gmail.com",
		Fields: []onepassword.Field{
			{ID: "username", Label: "username", Type: "STRING", Value: "me@gmail.com"},
			{ID: "password", Label: "password", Type: "CONCEALED", Value: "s3cret"},
			{ID: "empty", Label: "empty", Type: "STRING"},
			{ID: "TOTP_1", Label: "one-time password", Type: "OTP", Value: "otpauth://x"},
			{ID: "multi", Label: "address", Type: "STRING", Value: "line 1\nline 2"},
			{ID: "notesPlain", Label: "notesPlain", Type: "STRING", Purpose: "NOTES", Value: "hi"},
		},
	}

	row := func(key, name, value, raw string) map[string]string {
		return map[string]string{"name.text": name, "value.text": value, "raw.text": raw, "key.text": key}
	}
	want := []map[string]string{
		row("username", "username", "me@gmail.com", "me@gmail.com"),
		row("password", "password", Mask, "s3cret"),
		row("multi", "address", "line 1 …", "line 1\nline 2"),
		row("website", "website", "https://gmail.com", "https://gmail.com"),
	}
	if got := DetailRows(item); !reflect.DeepEqual(got, want) {
		t.Errorf("DetailRows =\n%v\nwant\n%v", got, want)
	}
}
