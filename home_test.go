package studio

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func completeHomeLinks() map[string]Link {
	return map[string]Link{
		"me":             {Href: "/me", Rel: "me", HTTPMethod: "GET"},
		"categories":     {Href: "/categories", Rel: "categories", HTTPMethod: "GET"},
		"artworks":       {Href: "/artworks", Rel: "artworks", HTTPMethod: "GET"},
		"artwork":        {Href: "/artworks/{id}", Rel: "artwork", HTTPMethod: "GET", Templated: true},
		"create-artwork": {Href: "/artworks", Rel: "create-artwork", HTTPMethod: "POST"},
	}
}

func homeWithout(rels ...string) map[string]Link {
	links := completeHomeLinks()
	for _, rel := range rels {
		delete(links, rel)
	}
	return links
}

func TestResolveHome(t *testing.T) {
	t.Parallel()
	want := HomeLinks{
		Me:            Link{Href: "/me", Rel: "me", HTTPMethod: "GET"},
		Categories:    Link{Href: "/categories", Rel: "categories", HTTPMethod: "GET"},
		Artworks:      Link{Href: "/artworks", Rel: "artworks", HTTPMethod: "GET"},
		Artwork:       Link{Href: "/artworks/{id}", Rel: "artwork", HTTPMethod: "GET", Templated: true},
		CreateArtwork: Link{Href: "/artworks", Rel: "create-artwork", HTTPMethod: "POST"},
	}
	tests := []struct {
		name    string
		home    Home
		want    HomeLinks
		wantErr string
	}{
		{
			name: "returns every link the home promises",
			home: Home{Links: completeHomeLinks()},
			want: want,
		},
		{
			name:    "fails when the me link is missing",
			home:    Home{Links: homeWithout("me")},
			wantErr: `home document missing link "me"`,
		},
		{
			name:    "fails when the categories link is missing",
			home:    Home{Links: homeWithout("categories")},
			wantErr: `home document missing link "categories"`,
		},
		{
			name:    "fails when the artworks link is missing",
			home:    Home{Links: homeWithout("artworks")},
			wantErr: `home document missing link "artworks"`,
		},
		{
			name:    "fails when the artwork link is missing",
			home:    Home{Links: homeWithout("artwork")},
			wantErr: `home document missing link "artwork"`,
		},
		{
			name:    "fails when the create-artwork link is missing",
			home:    Home{Links: homeWithout("create-artwork")},
			wantErr: `home document missing link "create-artwork"`,
		},
		{
			name:    "reports the first missing link in home order",
			home:    Home{Links: homeWithout("artworks", "artwork", "create-artwork")},
			wantErr: `home document missing link "artworks"`,
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
			if links != tt.want {
				t.Errorf("HomeLinks = %+v, want %+v", links, tt.want)
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
	if got, want := len(home.Links), 5; got != want {
		t.Errorf("len(Links) = %d, want %d", got, want)
	}
	links, err := ResolveHome(home)
	if err != nil {
		t.Fatalf("ResolveHome() error = %v, want nil", err)
	}
	want := HomeLinks{
		Me:            Link{Href: "/me", Rel: "me", HTTPMethod: "GET"},
		Categories:    Link{Href: "/categories", Rel: "categories", HTTPMethod: "GET"},
		Artworks:      Link{Href: "/artworks", Rel: "artworks", HTTPMethod: "GET"},
		Artwork:       Link{Href: "/artworks/{id}", Rel: "artwork", HTTPMethod: "GET", Templated: true},
		CreateArtwork: Link{Href: "/artworks", Rel: "create-artwork", HTTPMethod: "POST"},
	}
	if links != want {
		t.Errorf("HomeLinks = %+v, want %+v", links, want)
	}
	if !links.Artwork.Templated {
		t.Error("Artwork link is not templated, want a templated link")
	}
	if _, ok := home.Links["self"]; ok {
		t.Error(`home has a "self" link, want none`)
	}
}
