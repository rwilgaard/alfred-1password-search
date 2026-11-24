// src/internal/onepassword/onepassword_test.go
package onepassword_test

import (
	"os/exec"
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
)

func mockOK(output string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", output)
	}
}

func mockFail() func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
}

func TestNewService(t *testing.T) {
	_, err := onepassword.NewService("")
	if err == nil {
		t.Fatal("expected error for empty binPath")
	}
	svc, err := onepassword.NewService("op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestIsUnlocked(t *testing.T) {
	tests := []struct {
		name string
		mock func(string, ...string) *exec.Cmd
		want bool
	}{
		{
			name: "is unlocked",
			mock: mockOK(`{}`),
			want: true,
		},
		{
			name: "is locked",
			mock: mockFail(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &onepassword.Service{BinPath: "op", ExecCommand: tt.mock}
			if got := svc.IsUnlocked(); got != tt.want {
				t.Errorf("IsUnlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListItems(t *testing.T) {
	const mockJSON = `[{"id":"abc123","title":"Gmail","category":"LOGIN","vault":{"id":"vlt1","name":"Personal"},"additional_information":"user@gmail.com","urls":[{"href":"https://gmail.com","primary":true}]}]`

	tests := []struct {
		name      string
		vault     string
		mock      func(string, ...string) *exec.Cmd
		wantLen   int
		wantErr   bool
		wantFirst onepassword.Item
	}{
		{
			name:    "lists all items",
			vault:   "",
			mock:    mockOK(mockJSON),
			wantLen: 1,
			wantFirst: onepassword.Item{
				ID:         "abc123",
				Title:      "Gmail",
				Category:   "LOGIN",
				VaultID:    "vlt1",
				VaultName:  "Personal",
				PrimaryURL: "https://gmail.com",
			},
		},
		{
			name:    "empty vault returns all",
			vault:   "Personal",
			mock:    mockOK(mockJSON),
			wantLen: 1,
		},
		{
			name:    "command failure returns error",
			mock:    mockFail(),
			wantErr: true,
		},
		{
			name:    "empty list returns empty slice",
			mock:    mockOK(`[]`),
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &onepassword.Service{BinPath: "op", ExecCommand: tt.mock}
			items, err := svc.ListItems(tt.vault)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ListItems() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(items) != tt.wantLen {
				t.Fatalf("got %d items, want %d", len(items), tt.wantLen)
			}
			if tt.wantLen > 0 && tt.wantFirst != (onepassword.Item{}) && items[0] != tt.wantFirst {
				t.Errorf("got %+v, want %+v", items[0], tt.wantFirst)
			}
		})
	}
}

func TestGetItem(t *testing.T) {
	const mockJSON = `{"id":"abc123","title":"Gmail","category":"LOGIN","vault":{"id":"vlt1","name":"Personal"},"fields":[{"id":"username","label":"username","type":"STRING","purpose":"USERNAME","value":"user@gmail.com"},{"id":"password","label":"password","type":"CONCEALED","purpose":"PASSWORD","value":"s3cret"},{"id":"notesPlain","label":"notesPlain","type":"STRING","purpose":"NOTES","value":"some notes"},{"id":"TOTP_x","label":"one-time password","type":"OTP","value":"otpauth://totp/x?secret=ABC","totp":"123456"}],"urls":[{"href":"https://gmail.com","primary":true}]}`

	tests := []struct {
		name    string
		mock    func(string, ...string) *exec.Cmd
		wantErr bool
		check   func(*testing.T, *onepassword.ItemDetail)
	}{
		{
			name: "returns full item with fields",
			mock: mockOK(mockJSON),
			check: func(t *testing.T, item *onepassword.ItemDetail) {
				if item.ID != "abc123" {
					t.Errorf("ID = %q, want abc123", item.ID)
				}
				if item.PrimaryURL != "https://gmail.com" {
					t.Errorf("PrimaryURL = %q", item.PrimaryURL)
				}
				if len(item.Fields) != 4 {
					t.Fatalf("got %d fields, want 4", len(item.Fields))
				}
				if item.Fields[3].TOTP != "123456" {
					t.Errorf("OTP totp = %q, want 123456", item.Fields[3].TOTP)
				}
				if item.Fields[1].Type != "CONCEALED" {
					t.Errorf("password type = %q, want CONCEALED", item.Fields[1].Type)
				}
				if item.Fields[1].Value != "s3cret" {
					t.Errorf("password value = %q, want s3cret", item.Fields[1].Value)
				}
			},
		},
		{
			name:    "command failure returns error",
			mock:    mockFail(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &onepassword.Service{BinPath: "op", ExecCommand: tt.mock}
			item, err := svc.GetItem("abc123")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetItem() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, item)
			}
		})
	}
}

func TestLookupAndNotes(t *testing.T) {
	item := &onepassword.ItemDetail{Fields: []onepassword.Field{
		{Label: "login", Purpose: "USERNAME", Value: "me"},
		{Label: "renamed", Type: "CONCEALED", Purpose: "PASSWORD", Value: "s3cret"},
		{Label: "one-time password", Type: "OTP", Value: "otpauth://x", TOTP: "123456"},
		{Label: "notesPlain", Purpose: "NOTES", Value: "hi"},
	}}

	tests := []struct {
		key, want string
		found     bool
	}{
		{onepassword.KeyUsername, "me", true},
		{onepassword.KeyPassword, "s3cret", true},
		{onepassword.KeyOTP, "otpauth://x", true},
		{"other", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			f, ok := item.Lookup(tt.key)
			if ok != tt.found || f.Value != tt.want {
				t.Errorf("Lookup(%q) = %q, %v; want %q, %v", tt.key, f.Value, ok, tt.want, tt.found)
			}
		})
	}
	if got := item.Notes(); got != "hi" {
		t.Errorf("Notes() = %q, want hi", got)
	}
	if got := (&onepassword.ItemDetail{}).Notes(); got != "" {
		t.Errorf("Notes() on empty item = %q", got)
	}
}

func TestListVaults(t *testing.T) {
	const mockJSON = `[{"id":"vlt1","name":"Personal"},{"id":"vlt2","name":"Work"}]`

	svc := &onepassword.Service{BinPath: "op", ExecCommand: mockOK(mockJSON)}
	vaults, err := svc.ListVaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vaults) != 2 {
		t.Fatalf("got %d vaults, want 2", len(vaults))
	}
	if vaults[0].Name != "Personal" {
		t.Errorf("got %q, want Personal", vaults[0].Name)
	}
}

func TestGetAccountID(t *testing.T) {
	const mockJSON = `[{"url":"my.1password.com","email":"user@example.com","user_uuid":"USER1","account_uuid":"ACCT1"}]`

	svc := &onepassword.Service{BinPath: "op", ExecCommand: mockOK(mockJSON)}
	id, err := svc.GetAccountID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "ACCT1" {
		t.Errorf("got %q, want ACCT1", id)
	}

	svc2 := &onepassword.Service{BinPath: "op", ExecCommand: mockOK(`[]`)}
	_, err = svc2.GetAccountID()
	if err == nil {
		t.Fatal("expected error for empty accounts")
	}
}
