// Package usage keeps a frecency score per item: each use adds 1, and scores
// halve every HalfLife, so items used often and recently rank highest.
package usage

import (
	"math"
	"time"
)

// HalfLife is how long it takes a score to halve without use.
const HalfLife = 7 * 24 * time.Hour

// pruneBelow drops entries whose score has decayed below this (a single use
// after about a month), which keeps the file small.
const pruneBelow = 0.05

type entry struct {
	Score  float64 `json:"s"`
	UsedAt int64   `json:"t"` // unix seconds of the last use
}

// Store maps item ids to their usage. Its zero value is not usable; use make or a literal.
type Store map[string]entry

func (e entry) decayed(now time.Time) float64 {
	age := now.Sub(time.Unix(e.UsedAt, 0))
	return e.Score * math.Exp2(-age.Hours()/HalfLife.Hours())
}

// Record counts a use of id at now and prunes entries that have faded out.
func (s Store) Record(id string, now time.Time) {
	for k, e := range s {
		if k != id && e.decayed(now) < pruneBelow {
			delete(s, k)
		}
	}
	s[id] = entry{Score: s[id].decayed(now) + 1, UsedAt: now.Unix()}
}

// Scores returns the current score for each item.
func (s Store) Scores(now time.Time) map[string]float64 {
	out := make(map[string]float64, len(s))
	for id, e := range s {
		out[id] = e.decayed(now)
	}
	return out
}

// Latest returns the most recently used item.
func (s Store) Latest() (id string, usedAt time.Time, ok bool) {
	var latest int64
	for k, e := range s {
		if e.UsedAt > latest {
			id, latest = k, e.UsedAt
		}
	}
	return id, time.Unix(latest, 0), id != ""
}
