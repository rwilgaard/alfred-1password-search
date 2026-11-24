package util

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwilgaard/go-alfredutils/v2/alfred"
	"github.com/sethvargo/go-password/password"
)

var (
	IconFolder = alfred.Icon{Path: "icons/vault.png"}
	IconUpdate = alfred.Icon{Path: "icons/update.png"}
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

// IsCacheStale compares the 1Password app's database mtime with the last
// sync. Without access to the database it falls back to a 2 hour age.
func IsCacheStale(cache *alfred.Store, cacheKey string) bool {
	age, err := cache.Age(cacheKey)
	if err != nil {
		return true
	}
	dbTime, err := dbModTime()
	if err != nil {
		return age > 120*time.Minute
	}
	var lastSync int64
	if err := cache.LoadJSON(lastSyncDBTimeKey, &lastSync); err == nil {
		return dbTime.UnixNano() > lastSync
	}
	return time.Since(dbTime) < age
}

func SaveLastSyncDbTime(cache *alfred.Store) {
	if t, err := dbModTime(); err == nil {
		_ = cache.StoreJSON(lastSyncDBTimeKey, t.UnixNano()) // best effort: staleness falls back to the cache mtime
	}
}

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

// MatchesVault matches by name or ID, ignoring case. No targets match everything.
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
