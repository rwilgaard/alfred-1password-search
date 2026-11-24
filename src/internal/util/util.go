// src/internal/util/util.go
package util

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/sethvargo/go-password/password"
)

var IconFolder = &aw.Icon{Value: "icons/vault.png"}

func GeneratePassword(length int, symbols bool, allowedSymbols string) (string, error) {
	input := password.GeneratorInput{Symbols: allowedSymbols}
	sc := 0
	if symbols {
		sc = length / 4
	}
	gen, err := password.NewGenerator(&input)
	if err != nil {
		return "", err
	}
	return gen.Generate(length, length/4, sc, false, true)
}

func IsCacheStale(wf *aw.Workflow, cacheKey string) bool {
	cachePath := filepath.Join(wf.CacheDir(), cacheKey)
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return true
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}

	sqlitePath := filepath.Join(home, "Library/Group Containers/2BUA8C4S2C.com.1password/Library/Application Support/1Password/Data/1password.sqlite")
	dbInfo, err := os.Stat(sqlitePath)
	if err != nil {
		// Fallback to time-based check if SQLite database is not accessible
		return time.Since(cacheInfo.ModTime()) > 120*time.Minute
	}

	var lastSyncDbTime int64
	lastSyncDbTimeKey := "last_sync_db_time.json"
	if wf.Cache.Exists(lastSyncDbTimeKey) {
		if err := wf.Cache.LoadJSON(lastSyncDbTimeKey, &lastSyncDbTime); err == nil {
			return dbInfo.ModTime().UnixNano() > lastSyncDbTime
		}
	}

	return dbInfo.ModTime().After(cacheInfo.ModTime())
}

func SaveLastSyncDbTime(wf *aw.Workflow) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	sqlitePath := filepath.Join(home, "Library/Group Containers/2BUA8C4S2C.com.1password/Library/Application Support/1Password/Data/1password.sqlite")
	dbInfo, err := os.Stat(sqlitePath)
	if err != nil {
		return
	}

	_ = wf.Cache.StoreJSON("last_sync_db_time.json", dbInfo.ModTime().UnixNano())
}

// ParseVaults parses a slice of vault strings (which may be comma-separated)
// into a clean list of trimmed, non-empty vault identifiers.
func ParseVaults(inputs []string) []string {
	var result []string
	for _, input := range inputs {
		for _, part := range strings.Split(input, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

// MatchesVault checks whether the given item vault name or ID matches any of the target vaults.
// If targetVaults is empty, it returns true (matches any vault).
// Comparison is case-insensitive.
func MatchesVault(vaultName, vaultID string, targetVaults []string) bool {
	if len(targetVaults) == 0 {
		return true
	}
	for _, v := range targetVaults {
		if strings.EqualFold(vaultName, v) || strings.EqualFold(vaultID, v) {
			return true
		}
	}
	return false
}

func HasAll(input string, words []string) bool {
	for _, w := range words {
		if strings.Contains(input, w) {
			continue
		}
		return false
	}
	return true
}
