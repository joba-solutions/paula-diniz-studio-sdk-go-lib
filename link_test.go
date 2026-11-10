package studio

import "testing"

func TestExpand_ReplacesPlaceholders(t *testing.T) {
	t.Parallel()
	link := Link{Href: "/artworks/{id}/images/{imageId}", Rel: "image", HTTPMethod: "GET", Templated: true}
	expanded := Expand(link, map[string]string{"id": "01ARZ3NDEKTSV4RRFFQ69G5F02", "imageId": "media/1"})
	if got, want := expanded.Href, "/artworks/01ARZ3NDEKTSV4RRFFQ69G5F02/images/media%2F1"; got != want {
		t.Errorf("Href = %q, want %q", got, want)
	}
	if expanded.Templated {
		t.Error("Templated = true, want false after expansion")
	}
	if got, want := expanded.Rel, "image"; got != want {
		t.Errorf("Rel = %q, want %q", got, want)
	}
}

func TestWithQuery_AppendsAndSkipsEmptyValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		link  Link
		query map[string]string
		want  string
	}{
		{
			name:  "skips empty values",
			link:  Link{Href: "/artworks"},
			query: map[string]string{"published": "", "categoryId": "animals"},
			want:  "/artworks?categoryId=animals",
		},
		{
			name:  "appends to an existing query",
			link:  Link{Href: "/artworks?limit=2"},
			query: map[string]string{"cursor": "abc"},
			want:  "/artworks?limit=2&cursor=abc",
		},
		{
			name:  "returns the link when every value is empty",
			link:  Link{Href: "/artworks?limit=2"},
			query: map[string]string{"cursor": ""},
			want:  "/artworks?limit=2",
		},
		{
			name:  "encodes the values",
			link:  Link{Href: "/artworks"},
			query: map[string]string{"cursor": "a+b/c"},
			want:  "/artworks?cursor=a%2Bb%2Fc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WithQuery(tt.link, tt.query)
			if got.Href != tt.want {
				t.Errorf("Href = %q, want %q", got.Href, tt.want)
			}
		})
	}
}
