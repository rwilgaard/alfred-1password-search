// src/internal/onepassword/onepassword.go
package onepassword

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

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
	ID         string
	Title      string
	Category   string
	VaultID    string
	VaultName  string
	PrimaryURL string
	Fields     []Field
}

type Field struct {
	ID      string
	Label   string
	Type    string
	Value   string
	Purpose string
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

type opItemList struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	Category              string  `json:"category"`
	Vault                 opVault `json:"vault"`
	URLs                  []opURL `json:"urls"`
	AdditionalInformation string  `json:"additional_information"`
}

type opItemDetail struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Category string    `json:"category"`
	Vault    opVault   `json:"vault"`
	Fields   []opField `json:"fields"`
	URLs     []opURL   `json:"urls"`
}

type opAccount struct {
	AccountUUID string `json:"account_uuid"`
}

func NewService(binPath string) (*Service, error) {
	if binPath == "" {
		return nil, errors.New("binPath is empty")
	}
	return &Service{
		BinPath:     binPath,
		ExecCommand: exec.Command,
	}, nil
}

func (s *Service) IsUnlocked() bool {
	cmd := s.ExecCommand(s.BinPath, "user", "get", "--me")
	err := cmd.Run()
	return err == nil
}

func (s *Service) ListItems(vaultName string) ([]Item, error) {
	args := []string{"item", "list", "--format", "json"}
	if vaultName != "" {
		args = append(args, "--vault", vaultName)
	}
	out, err := s.ExecCommand(s.BinPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("op item list: %w", err)
	}
	var raw []opItemList
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse op item list: %w", err)
	}
	items := make([]Item, len(raw))
	for i, r := range raw {
		items[i] = Item{
			ID:         r.ID,
			Title:      r.Title,
			Category:   r.Category,
			VaultID:    r.Vault.ID,
			VaultName:  r.Vault.Name,
			PrimaryURL: primaryURL(r.URLs),
		}
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
	return &ItemDetail{
		ID:         raw.ID,
		Title:      raw.Title,
		Category:   raw.Category,
		VaultID:    raw.Vault.ID,
		VaultName:  raw.Vault.Name,
		PrimaryURL: primaryURL(raw.URLs),
		Fields:     fields,
	}, nil
}

func (s *Service) ListVaults() ([]Vault, error) {
	out, err := s.ExecCommand(s.BinPath, "vault", "list", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("op vault list: %w", err)
	}
	var raw []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse op vault list: %w", err)
	}
	vaults := make([]Vault, len(raw))
	for i, v := range raw {
		vaults[i] = Vault{ID: v.ID, Name: v.Name}
	}
	return vaults, nil
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
