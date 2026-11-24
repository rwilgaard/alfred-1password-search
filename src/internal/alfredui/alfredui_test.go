package alfredui_test

import (
	"bytes"
	"testing"

	"github.com/rwilgaard/alfred-1password-search/src/internal/alfredui"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name string
		a    alfredui.Actions
		want string
	}{
		{
			name: "props merge on same id",
			a:    alfredui.NewUpdate().Status("status", "bad", alfredui.ColourError).Enable(false, "status"),
			want: `{"update":{"ids":{"status":{"colour":"systemRedColor","enabled":false,"text":"bad"}}}}`,
		},
		{
			name: "finish",
			a:    alfredui.Actions{Finish: &alfredui.Finish{Args: "ok", Variables: map[string]string{"k": "v"}}},
			want: `{"finish":{"args":"ok","variables":{"k":"v"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := alfredui.Write(&b, tt.a); err != nil {
				t.Fatal(err)
			}
			if got := b.String(); got != tt.want+"\n" {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}
