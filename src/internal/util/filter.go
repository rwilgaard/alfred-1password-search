package util

import (
	"cmp"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"go.deanishe.net/fuzzy"
)

type scoredItem struct {
	item  onepassword.Item
	score float64
}

// scoreField evaluates how well a field matches a search term.
func scoreField(val, term string, exactBonus, prefixBonus, containsBonus, fuzzyBase float64, isFuzzy bool) (float64, bool) {
	if val == "" || term == "" {
		return 0, false
	}
	vLower := strings.ToLower(val)
	tLower := strings.ToLower(term)

	var score float64
	var matched bool

	if vLower == tLower {
		return exactBonus, true
	}
	if strings.HasPrefix(vLower, tLower) {
		score += prefixBonus
		matched = true
	} else if strings.Contains(vLower, tLower) {
		score += containsBonus
		matched = true
	}

	if isFuzzy {
		res := fuzzy.Match(val, term)
		if res.Match {
			score += fuzzyBase + res.Score
			matched = true
		}
	}

	return score, matched
}

// scoreItemWord evaluates how well an item matches a single search word across all its fields.
func scoreItemWord(it onepassword.Item, word string, isFuzzy bool) (float64, bool) {
	var wordScore float64
	var wordMatched bool

	// Title: highest priority (exact: 1000, prefix: 500, contains: 200, fuzzy: 100)
	if s, m := scoreField(it.Title, word, 1000, 500, 200, 100, isFuzzy); m {
		wordScore += s
		wordMatched = true
	}
	// ID: high priority (exact: 800, prefix: 400, contains: 150, fuzzy: 50)
	if s, m := scoreField(it.ID, word, 800, 400, 150, 50, isFuzzy); m {
		wordScore += s
		wordMatched = true
	}
	// URL: lower priority (exact: 100, prefix: 50, contains: 30, fuzzy: 10)
	if s, m := scoreField(it.PrimaryURL, word, 100, 50, 30, 10, isFuzzy); m {
		wordScore += s
		wordMatched = true
	}
	// VaultName: lower priority (exact: 50, prefix: 30, contains: 20, fuzzy: 5)
	if s, m := scoreField(it.VaultName, word, 50, 30, 20, 5, isFuzzy); m {
		wordScore += s
		wordMatched = true
	}
	// Category: lower priority (exact: 50, prefix: 30, contains: 20, fuzzy: 5)
	if s, m := scoreField(it.Category, word, 50, 30, 20, 5, isFuzzy); m {
		wordScore += s
		wordMatched = true
	}

	return wordScore, wordMatched
}

// ScoreItem computes an aggregate relevance score for an item against a query.
// For multi-word queries, every word must match at least one field of the item.
func ScoreItem(it onepassword.Item, query string, isFuzzy bool) (float64, bool) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return 0, true
	}

	var totalScore float64

	// If multiple words, grant additional phrase bonus if the full query matches Title or URL
	if len(words) > 1 {
		if s, m := scoreField(it.Title, query, 2000, 1000, 400, 200, isFuzzy); m {
			totalScore += s
		}
	}

	for _, w := range words {
		s, matched := scoreItemWord(it, w, isFuzzy)
		if !matched {
			return 0, false
		}
		totalScore += s
	}

	return totalScore, true
}

// frecencyWeight scales the usage boost. Heavy use (score ~50) adds ~280, enough
// to reorder items within a match tier but not to lift a contains match over a
// prefix match (a 300 gap).
const frecencyWeight = 50

// FilterAndRankItems filters and ranks items based on how well their fields match the query.
// Items with matching Title and ID rank significantly higher than items matching URL, VaultName, or Category.
// usage holds frecency scores by item ID (may be nil); with an empty query items are ordered by it.
func FilterAndRankItems(items []onepassword.Item, query string, isFuzzy bool, usage map[string]float64) []onepassword.Item {
	query = strings.TrimSpace(query)
	if query == "" {
		if len(usage) == 0 {
			return items
		}
		sorted := slices.Clone(items)
		slices.SortStableFunc(sorted, func(a, b onepassword.Item) int {
			return cmp.Compare(usage[b.ID], usage[a.ID])
		})
		return sorted
	}

	var scored []scoredItem
	for _, item := range items {
		if s, matched := ScoreItem(item, query, isFuzzy); matched {
			s += frecencyWeight * math.Log2(1+usage[item.ID])
			scored = append(scored, scoredItem{item: item, score: s})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return strings.ToLower(scored[i].item.Title) < strings.ToLower(scored[j].item.Title)
	})

	result := make([]onepassword.Item, len(scored))
	for i, s := range scored {
		result[i] = s.item
	}
	return result
}
