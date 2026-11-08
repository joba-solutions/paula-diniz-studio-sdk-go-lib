package studio

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestResolveHome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		home    Home
		want    Link
		wantErr string
	}{
		{
			name: "returns the me link",
			home: Home{Links: map[string]Link{
				"me": {Href: "/me", Rel: "me", HTTPMethod: "GET"},
			}},
			want: Link{Href: "/me", Rel: "me", HTTPMethod: "GET"},
		},
		{
			name:    "fails when the me link is missing",
			home:    Home{Links: map[string]Link{}},
			wantErr: `home document missing link "me"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			links, err := ResolveHome(tt.home)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ResolveHome() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveHome() error = %v, want nil", err)
			}
			if links.Me != tt.want {
				t.Errorf("Me = %+v, want %+v", links.Me, tt.want)
			}
		})
	}
}

func TestResolveHome_GoldenDocument(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/home.json")
	if err != nil {
		t.Fatalf("read testdata/home.json: %v", err)
	}
	var home Home
	if err := json.Unmarshal(data, &home); err != nil {
		t.Fatalf("decode testdata/home.json: %v", err)
	}
	if got, want := len(home.Links), 1; got != want {
		t.Errorf("len(Links) = %d, want %d", got, want)
	}
	links, err := ResolveHome(home)
	if err != nil {
		t.Fatalf("ResolveHome() error = %v, want nil", err)
	}
	want := Link{Href: "/me", Rel: "me", HTTPMethod: "GET"}
	if links.Me != want {
		t.Errorf("Me = %+v, want %+v", links.Me, want)
	}
	if _, ok := home.Links["self"]; ok {
		t.Error(`home has a "self" link, want none`)
	}
}
