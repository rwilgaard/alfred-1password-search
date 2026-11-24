package util_test

import (
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
)

func TestFilterAndRankItems(t *testing.T) {
	items := []onepassword.Item{
		{
			ID:         "id-github",
			Title:      "GitHub",
			PrimaryURL: "https://github.com",
			VaultName:  "Personal",
			Category:   "Login",
		},
		{
			ID:         "id-ghe",
			Title:      "GitHub Enterprise",
			PrimaryURL: "https://ghe.company.com",
			VaultName:  "Work",
			Category:   "Login",
		},
		{
			ID:         "id-resources",
			Title:      "Developer Resources",
			PrimaryURL: "https://github.com/resources",
			VaultName:  "Personal",
			Category:   "Login",
		},
		{
			ID:         "id-special-aws",
			Title:      "Some Cloud Service",
			PrimaryURL: "https://example.com",
			VaultName:  "Personal",
			Category:   "Login",
		},
		{
			ID:         "id-aws",
			Title:      "AWS Management Console",
			PrimaryURL: "https://aws.amazon.com",
			VaultName:  "Work",
			Category:   "Login",
		},
	}

	t.Run("title match ranks above url match", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "github", true, nil)
		if len(res) < 3 {
			t.Fatalf("expected at least 3 matches, got %d", len(res))
		}
		// "GitHub" should be #1 because of exact title match
		if res[0].Title != "GitHub" {
			t.Errorf("expected res[0] to be 'GitHub', got %q", res[0].Title)
		}
		// "GitHub Enterprise" should rank above "Developer Resources" (which only has github in URL)
		if res[1].Title != "GitHub Enterprise" {
			t.Errorf("expected res[1] to be 'GitHub Enterprise', got %q", res[1].Title)
		}
		if res[2].Title != "Developer Resources" {
			t.Errorf("expected res[2] to be 'Developer Resources', got %q", res[2].Title)
		}
	})

	t.Run("exact id finds the item", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "ID-SPECIAL-AWS", true, nil)
		if len(res) != 1 || res[0].Title != "Some Cloud Service" {
			t.Fatalf("got %+v, want only 'Some Cloud Service'", res)
		}
	})

	t.Run("partial id does not match", func(t *testing.T) {
		// IDs are random strings, so partial and fuzzy hits on them are noise.
		res := util.FilterAndRankItems(items, "special", true, nil)
		if len(res) != 0 {
			t.Fatalf("got %d matches, want 0", len(res))
		}
	})

	t.Run("multi-word query across title and vault", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "github work", true, nil)
		if len(res) != 1 {
			t.Fatalf("expected exactly 1 match ('GitHub Enterprise' in Work vault), got %d", len(res))
		}
		if res[0].Title != "GitHub Enterprise" {
			t.Errorf("expected 'GitHub Enterprise', got %q", res[0].Title)
		}
	})

	t.Run("empty query returns all items in original order", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "", true, nil)
		if len(res) != len(items) {
			t.Fatalf("expected %d items, got %d", len(items), len(res))
		}
	})

	t.Run("empty query orders by usage, unused keep their order", func(t *testing.T) {
		usage := map[string]float64{"id-aws": 3, "id-resources": 1}
		res := util.FilterAndRankItems(items, "", true, usage)
		want := []string{"id-aws", "id-resources", "id-github", "id-ghe", "id-special-aws"}
		for i, id := range want {
			if res[i].ID != id {
				t.Fatalf("res[%d] = %q, want %q", i, res[i].ID, id)
			}
		}
		if items[0].ID != "id-github" {
			t.Error("input slice was reordered")
		}
	})

	t.Run("usage reorders within a match tier", func(t *testing.T) {
		// Both titles start with "github"; heavy use of the Enterprise entry lifts it.
		res := util.FilterAndRankItems(items, "githu", false, map[string]float64{"id-ghe": 20})
		if res[0].ID != "id-ghe" {
			t.Errorf("res[0] = %q, want id-ghe", res[0].ID)
		}
	})

	t.Run("usage does not beat a stronger match", func(t *testing.T) {
		// Exact title match outranks a heavily used URL-only match.
		res := util.FilterAndRankItems(items, "github", true, map[string]float64{"id-resources": 50})
		if res[0].ID != "id-github" {
			t.Errorf("res[0] = %q, want id-github", res[0].ID)
		}
	})

	t.Run("non-matching query returns empty", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "nonexistent-query-xyz", true, nil)
		if len(res) != 0 {
			t.Fatalf("expected 0 items, got %d", len(res))
		}
	})

	t.Run("non-fuzzy search matches substrings with field ranking", func(t *testing.T) {
		res := util.FilterAndRankItems(items, "github", false, nil)
		want := []string{"GitHub", "GitHub Enterprise", "Developer Resources"}
		if len(res) != len(want) {
			t.Fatalf("expected %d matches for 'github', got %d", len(want), len(res))
		}
		for i, title := range want {
			if res[i].Title != title {
				t.Errorf("res[%d] = %q, want %q", i, res[i].Title, title)
			}
		}
	})
}
