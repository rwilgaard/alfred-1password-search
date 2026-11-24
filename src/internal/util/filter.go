package util

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"go.deanishe.net/fuzzy"
)

// weights are the bonuses for an exact, prefix, contains and fuzzy match on a field.
type weights struct{ exact, prefix, contains, fuzzy float64 }

var (
	titleWeights  = weights{1000, 500, 200, 100}
	urlWeights    = weights{100, 50, 30, 10}
	metaWeights   = weights{50, 30, 20, 5} // vault name and category
	phraseWeights = weights{2000, 1000, 400, 200}
)

// idExactBonus rewards pasting an item ID. IDs only match exactly: they are
// random, so partial or fuzzy matches on them are noise.
const idExactBonus = 800

// scoreField scores val against term, which must already be lower case.
func scoreField(val, term string, w weights, isFuzzy bool) (float64, bool) {
	if val == "" {
		return 0, false
	}
	vLower := strings.ToLower(val)
	if vLower == term {
		return w.exact, true
	}

	var score float64
	var matched bool
	if strings.HasPrefix(vLower, term) {
		score, matched = w.prefix, true
	} else if strings.Contains(vLower, term) {
		score, matched = w.contains, true
	}
	if isFuzzy {
		if res := fuzzy.Match(val, term); res.Match {
			score += w.fuzzy + res.Score
			matched = true
		}
	}
	return score, matched
}

// scoreItemWord scores one lower-case query word against all of an item's fields.
func scoreItemWord(it onepassword.Item, word string, isFuzzy bool) (float64, bool) {
	var total float64
	var matched bool
	for _, f := range [...]struct {
		val string
		w   weights
	}{
		{it.Title, titleWeights},
		{it.PrimaryURL, urlWeights},
		{it.VaultName, metaWeights},
		{it.Category, metaWeights},
	} {
		if s, m := scoreField(f.val, word, f.w, isFuzzy); m {
			total += s
			matched = true
		}
	}
	if strings.EqualFold(it.ID, word) {
		total += idExactBonus
		matched = true
	}
	return total, matched
}

// scoreItem scores an item against a lower-case query split into words.
// Every word must match at least one field of the item.
func scoreItem(it onepassword.Item, query string, words []string, isFuzzy bool) (float64, bool) {
	var total float64
	// A multi-word query also earns a bonus when the whole phrase matches the title.
	if len(words) > 1 {
		if s, m := scoreField(it.Title, query, phraseWeights, isFuzzy); m {
			total += s
		}
	}
	for _, w := range words {
		s, matched := scoreItemWord(it, w, isFuzzy)
		if !matched {
			return 0, false
		}
		total += s
	}
	return total, true
}

// frecencyWeight scales the usage boost. Heavy use (score ~50) adds ~280, enough
// to reorder items within a match tier but not to lift a contains match over a
// prefix match (a 300 gap).
const frecencyWeight = 50

// FilterAndRankItems returns the items matching query, best first: title
// matches rank well above URL, vault and category matches, and usage
// (frecency scores by item ID, may be nil) breaks near-ties. With an empty
// query all items are returned, ordered by usage.
func FilterAndRankItems(items []onepassword.Item, query string, isFuzzy bool, usage map[string]float64) []onepassword.Item {
	query = strings.ToLower(strings.TrimSpace(query))
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

	type scored struct {
		item  onepassword.Item
		title string // lower-cased, for the tie-break
		score float64
	}
	words := strings.Fields(query)
	var matches []scored
	for _, item := range items {
		if s, ok := scoreItem(item, query, words, isFuzzy); ok {
			s += frecencyWeight * math.Log2(1+usage[item.ID])
			matches = append(matches, scored{item, strings.ToLower(item.Title), s})
		}
	}
	slices.SortFunc(matches, func(a, b scored) int {
		return cmp.Or(cmp.Compare(b.score, a.score), strings.Compare(a.title, b.title))
	})

	result := make([]onepassword.Item, len(matches))
	for i, m := range matches {
		result[i] = m.item
	}
	return result
}
