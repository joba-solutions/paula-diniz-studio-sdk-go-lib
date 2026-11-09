package studio

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestCategories_GoldenDocument(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/categories.json")
	if err != nil {
		t.Fatalf("read testdata/categories.json: %v", err)
	}
	var categories Categories
	if err := json.Unmarshal(data, &categories); err != nil {
		t.Fatalf("decode testdata/categories.json: %v", err)
	}
	want := []Category{
		{
			ID:   "animals",
			Slug: LocalizedText{PTBR: "animais", EN: "animals"},
			Name: LocalizedText{PTBR: "Animais", EN: "Animals"},
		},
		{
			ID:   "portraits",
			Slug: LocalizedText{PTBR: "retratos", EN: "portraits"},
			Name: LocalizedText{PTBR: "Retratos", EN: "Portraits"},
		},
		{
			ID:   "landscapes",
			Slug: LocalizedText{PTBR: "paisagens", EN: "landscapes"},
			Name: LocalizedText{PTBR: "Paisagens", EN: "Landscapes"},
		},
		{
			ID:   "murals",
			Slug: LocalizedText{PTBR: "murais", EN: "murals"},
			Name: LocalizedText{PTBR: "Murais", EN: "Murals"},
		},
		{
			ID:   "other",
			Slug: LocalizedText{PTBR: "outros", EN: "other"},
			Name: LocalizedText{PTBR: "Outros", EN: "Other works"},
		},
	}
	if !slices.Equal(categories.Categories, want) {
		t.Errorf("Categories = %+v, want %+v", categories.Categories, want)
	}
	wantLinks := CategoriesLinks{Self: Link{Href: "/categories", Rel: "self", HTTPMethod: "GET"}}
	if categories.Links != wantLinks {
		t.Errorf("Links = %+v, want %+v", categories.Links, wantLinks)
	}
}
