// Package onepassword wraps the op CLI.
package onepassword

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// Service runs op. ExecCommand is swappable so tests can stub the CLI.
type Service struct {
	BinPath     string
	ExecCommand func(name string, arg ...string) *exec.Cmd
}

type Item struct {
	ID         string
	Title      string
	Category   string
	VaultID    string
	VaultName  string
	PrimaryURL string
}

type ItemDetail struct {
	Item
	Fields []Field
}

type Field struct {
	ID      string
	Label   string
	Type    string
	Value   string
	Purpose string
}

const (
	KeyPassword = "password"
	KeyUsername = "username"
	KeyOTP      = "otp"
)

// Lookup returns the item's password, username or OTP field. It matches on
// purpose and type rather than label, since users can rename labels.
func (d *ItemDetail) Lookup(key string) (Field, bool) {
	for _, f := range d.Fields {
		if (key == KeyPassword && f.Purpose == "PASSWORD") ||
			(key == KeyUsername && f.Purpose == "USERNAME") ||
			(key == KeyOTP && f.Type == "OTP") {
			return f, true
		}
	}
	return Field{}, false
}

func (d *ItemDetail) Notes() string {
	for _, f := range d.Fields {
		if f.Purpose == "NOTES" {
			return f.Value
		}
	}
	return ""
}

type Vault struct {
	ID   string
	Name string
}

type opVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type opURL struct {
	Href    string `json:"href"`
	Primary bool   `json:"primary"`
}

type opField struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Purpose string `json:"purpose"`
}

type opItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Category string  `json:"category"`
	Vault    opVault `json:"vault"`
	URLs     []opURL `json:"urls"`
}

func (r opItem) item() Item {
	return Item{
		ID:         r.ID,
		Title:      r.Title,
		Category:   r.Category,
		VaultID:    r.Vault.ID,
		VaultName:  r.Vault.Name,
		PrimaryURL: primaryURL(r.URLs),
	}
}

type opItemDetail struct {
	opItem
	Fields []opField `json:"fields"`
}

type opAccount struct {
	AccountUUID string `json:"account_uuid"`
}

func NewService() *Service {
	return &Service{BinPath: "op", ExecCommand: exec.Command}
}

// IsAuthorized never prompts: op whoami fails instead. It can report locked
// when Alfred's processes are unlocked, so use it only to skip work.
func (s *Service) IsAuthorized() bool {
	return s.ExecCommand(s.BinPath, "whoami").Run() == nil
}

func (s *Service) Authorize() error {
	if err := s.ExecCommand(s.BinPath, "user", "get", "--me").Run(); err != nil {
		return fmt.Errorf("unlock 1Password: %w", err)
	}
	return nil
}

// ListItems returns items without their fields or secrets.
func (s *Service) ListItems() ([]Item, error) {
	out, err := s.ExecCommand(s.BinPath, "item", "list", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("op item list: %w", err)
	}
	var raw []opItem
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse op item list: %w", err)
	}
	items := make([]Item, len(raw))
	for i, r := range raw {
		items[i] = r.item()
	}
	return items, nil
}

func (s *Service) GetItem(id string) (*ItemDetail, error) {
	out, err := s.ExecCommand(s.BinPath, "item", "get", id, "--format", "json", "--reveal").Output()
	if err != nil {
		return nil, fmt.Errorf("op item get: %w", err)
	}
	var raw opItemDetail
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse op item get: %w", err)
	}
	fields := make([]Field, len(raw.Fields))
	for i, f := range raw.Fields {
		fields[i] = Field(f)
	}
	return &ItemDetail{Item: raw.item(), Fields: fields}, nil
}

func Vaults(items []Item) []Vault {
	seen := map[string]bool{}
	var vaults []Vault
	for _, it := range items {
		if !seen[it.VaultID] {
			seen[it.VaultID] = true
			vaults = append(vaults, Vault{ID: it.VaultID, Name: it.VaultName})
		}
	}
	slices.SortFunc(vaults, func(a, b Vault) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return vaults
}

func (s *Service) GetAccountID() (string, error) {
	out, err := s.ExecCommand(s.BinPath, "account", "list", "--format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("op account list: %w", err)
	}
	var accounts []opAccount
	if err := json.Unmarshal(out, &accounts); err != nil {
		return "", fmt.Errorf("parse op account list: %w", err)
	}
	if len(accounts) == 0 {
		return "", errors.New("no 1Password accounts found")
	}
	return accounts[0].AccountUUID, nil
}

func primaryURL(urls []opURL) string {
	for _, u := range urls {
		if u.Primary {
			return u.Href
		}
	}
	if len(urls) > 0 {
		return urls[0].Href
	}
	return ""
}
