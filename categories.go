package studio

// LocalizedText is a value in both site languages, as the API sends it.
type LocalizedText struct {
	PTBR string `json:"ptBR"`
	EN   string `json:"en"`
}

// Category is one of the Studio's fixed categories.
type Category struct {
	ID   string        `json:"id"`
	Slug LocalizedText `json:"slug"`
	Name LocalizedText `json:"name"`
}

// CategoriesLinks are the links on the categories collection.
type CategoriesLinks struct {
	Self Link `json:"self"`
}

// Categories is the fixed category list, as GET /categories returns it.
type Categories struct {
	Categories []Category      `json:"categories"`
	Links      CategoriesLinks `json:"links"`
}
