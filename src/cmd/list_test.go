package cmd

import (
	"bytes"
	"os/exec"
	"slices"
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/onepassword"
	"github.com/rwilgaard/go-alfredutils/v2/alfred"
	"github.com/rwilgaard/go-alfredutils/v2/alfred/alfredtest"
)

var testItems = []onepassword.Item{
	{ID: "id-github", Title: "GitHub", PrimaryURL: "https://github.com", VaultName: "Personal", Category: "LOGIN"},
	{ID: "id-ghe", Title: "GitHub Enterprise", PrimaryURL: "https://ghe.company.com", VaultName: "Work", Category: "LOGIN"},
	{ID: "id-resources", Title: "Developer Resources", PrimaryURL: "https://github.com/resources", VaultName: "Personal", Category: "LOGIN"},
	{ID: "id-aws", Title: "AWS Management Console", PrimaryURL: "https://aws.amazon.com", VaultName: "Work", Category: "LOGIN"},
}

// setupList points the package workflow at a fresh test one with a just
// written items cache, so checkSync starts no sync job.
func setupList(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out *bytes.Buffer
	wf, out = alfredtest.New(t, nil, alfred.WithUsage())
	cfg = &workflowConfig{FuzzySearch: true, ModifierReturn: "Copy Password"}
	if err := wf.Cache.StoreJSON(itemsCacheKey, testItems); err != nil {
		t.Fatal(err)
	}
	return out
}

func runList(t *testing.T, out *bytes.Buffer, query string) alfredtest.Feedback {
	t.Helper()
	listCmd.Run(listCmd, []string{query})
	return alfredtest.Decode(t, out)
}

func uids(fb alfredtest.Feedback) []string {
	var ids []string
	for _, it := range fb.Items {
		ids = append(ids, it.UID)
	}
	return ids
}

func TestListRanking(t *testing.T) {
	tests := []struct {
		name  string
		query string
		uses  []string
		want  []string
	}{
		{"title beats url", "github", nil, []string{"u:id-github", "u:id-ghe", "u:id-resources"}},
		{"words match across fields", "github work", nil, []string{"u:id-ghe"}},
		{"pasted id", "id-aws", nil, []string{"u:id-aws"}},
		{"usage reorders within a tier", "githu", []string{"id-ghe", "id-ghe", "id-ghe"}, []string{"u:id-ghe", "u:id-github", "u:id-resources"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := setupList(t)
			cfg.FuzzySearch = false
			for _, id := range tt.uses {
				recordUse(id)
			}
			fb := runList(t, out, tt.query)
			if got := uids(fb); !slices.Equal(got, tt.want) {
				t.Errorf("uids = %q, want %q", got, tt.want)
			}
			if !fb.SkipKnowledge {
				t.Error("skipknowledge not set")
			}
		})
	}
}

func TestListPinnedRows(t *testing.T) {
	t.Run("recent row first on an empty query", func(t *testing.T) {
		out := setupList(t)
		recordUse("id-aws")
		fb := runList(t, out, "")
		if fb.Items[0].UID != "u:recent" || fb.Items[0].Title != "↺ AWS Management Console" {
			t.Fatalf("first item = %q %q", fb.Items[0].UID, fb.Items[0].Title)
		}
		if fb.Items[0].Variables["action"] != actionShowDetails {
			t.Errorf("recent action = %q", fb.Items[0].Variables["action"])
		}
		if fb.Items[1].UID != "u:id-aws" {
			t.Errorf("most used item not next: %q", uids(fb))
		}
	})

	t.Run("no recent row while typing", func(t *testing.T) {
		out := setupList(t)
		recordUse("id-aws")
		if got := runList(t, out, "aws").Titles(); !slices.Equal(got, []string{"AWS Management Console"}) {
			t.Errorf("titles = %q", got)
		}
	})

	t.Run("sync status stays above matches", func(t *testing.T) {
		out := setupList(t)
		// Hold the sync job name so checkSync reports a running sync.
		if err := wf.RunInBackground(syncJobName, exec.Command("/bin/sleep", "5")); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = wf.Kill(syncJobName) })
		got := runList(t, out, "sync").Titles()
		want := []string{"Updating items…", "No results found..."}
		if !slices.Equal(got, want) {
			t.Errorf("titles = %q, want %q", got, want)
		}
	})
}
