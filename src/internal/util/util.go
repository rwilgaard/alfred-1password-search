// Package util holds the workflow's icons and helpers shared by the commands.
package util

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/sethvargo/go-password/password"
)

var (
	IconFolder = &aw.Icon{Value: "icons/vault.png"}
	IconUpdate = &aw.Icon{Value: "icons/update.png"}
)

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

const lastSyncDBTimeKey = "last_sync_db_time.json"

// dbModTime returns the mtime of the 1Password app's database, which changes
// whenever the vault data does.
func dbModTime() (time.Time, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return time.Time{}, err
	}
	fi, err := os.Stat(filepath.Join(home, "Library/Group Containers/2BUA8C4S2C.com.1password/Library/Application Support/1Password/Data/1password.sqlite"))
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

// IsCacheStale reports whether 1Password has changed since the cache was
// written. Without access to the database it falls back to a 2 hour age.
func IsCacheStale(wf *aw.Workflow, cacheKey string) bool {
	cacheInfo, err := os.Stat(filepath.Join(wf.CacheDir(), cacheKey))
	if err != nil {
		return true
	}
	dbTime, err := dbModTime()
	if err != nil {
		return time.Since(cacheInfo.ModTime()) > 120*time.Minute
	}
	var lastSync int64
	if err := wf.Cache.LoadJSON(lastSyncDBTimeKey, &lastSync); err == nil {
		return dbTime.UnixNano() > lastSync
	}
	return dbTime.After(cacheInfo.ModTime())
}

// SaveLastSyncDbTime records the database mtime the cache was synced at.
func SaveLastSyncDbTime(wf *aw.Workflow) {
	if t, err := dbModTime(); err == nil {
		_ = wf.Cache.StoreJSON(lastSyncDBTimeKey, t.UnixNano()) // best effort: staleness falls back to the cache mtime
	}
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
