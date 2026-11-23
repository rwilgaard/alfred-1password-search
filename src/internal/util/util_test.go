package util_test

import (
	"reflect"
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
)

func TestParseVaults(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "single vault",
			input:    []string{"Personal"},
			expected: []string{"Personal"},
		},
		{
			name:     "comma-separated in single string",
			input:    []string{"Personal, Work, Archive"},
			expected: []string{"Personal", "Work", "Archive"},
		},
		{
			name:     "multiple strings with whitespace and empty parts",
			input:    []string{" Personal , Work ", "", "   ", "Archive,   , Family "},
			expected: []string{"Personal", "Work", "Archive", "Family"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := util.ParseVaults(tc.input)
			if len(actual) == 0 && len(tc.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("ParseVaults(%v) = %v, expected %v", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestMatchesVault(t *testing.T) {
	tests := []struct {
		name         string
		vaultName    string
		vaultID      string
		targetVaults []string
		expected     bool
	}{
		{
			name:         "empty targets matches anything",
			vaultName:    "Personal",
			vaultID:      "id-123",
			targetVaults: []string{},
			expected:     true,
		},
		{
			name:         "nil targets matches anything",
			vaultName:    "Personal",
			vaultID:      "id-123",
			targetVaults: nil,
			expected:     true,
		},
		{
			name:         "exact name match",
			vaultName:    "Personal",
			vaultID:      "id-123",
			targetVaults: []string{"Personal"},
			expected:     true,
		},
		{
			name:         "case-insensitive name match",
			vaultName:    "Personal",
			vaultID:      "id-123",
			targetVaults: []string{"personal"},
			expected:     true,
		},
		{
			name:         "exact id match",
			vaultName:    "Personal",
			vaultID:      "id-123",
			targetVaults: []string{"id-123"},
			expected:     true,
		},
		{
			name:         "case-insensitive id match",
			vaultName:    "Personal",
			vaultID:      "ID-123",
			targetVaults: []string{"id-123"},
			expected:     true,
		},
		{
			name:         "match in multi-vault list",
			vaultName:    "Work",
			vaultID:      "id-456",
			targetVaults: []string{"Personal", "Work", "Archive"},
			expected:     true,
		},
		{
			name:         "no match",
			vaultName:    "Private",
			vaultID:      "id-789",
			targetVaults: []string{"Personal", "Work"},
			expected:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := util.MatchesVault(tc.vaultName, tc.vaultID, tc.targetVaults)
			if actual != tc.expected {
				t.Errorf("MatchesVault(%q, %q, %v) = %v, expected %v",
					tc.vaultName, tc.vaultID, tc.targetVaults, actual, tc.expected)
			}
		})
	}
}
