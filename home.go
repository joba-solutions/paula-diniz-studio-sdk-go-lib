package studio

import (
	"errors"
	"net/http"
)

// Home is the API root: the links a client starts from.
type Home struct {
	Links map[string]Link `json:"links"`
}

// HomeLink is the one path a client may know.
var HomeLink = Link{Href: "/", Rel: "home", HTTPMethod: http.MethodGet}

// HomeLinks are the home's links, resolved into the entry points a caller follows.
type HomeLinks struct {
	Me            Link
	Categories    Link
	Artworks      Link
	Artwork       Link
	CreateArtwork Link
}

// ResolveHome requires every link the home promises, failing when the server drops one.
func ResolveHome(home Home) (HomeLinks, error) {
	me, ok := home.Links["me"]
	if !ok {
		return HomeLinks{}, errors.New(`studio: home document missing link "me"`)
	}
	categories, ok := home.Links["categories"]
	if !ok {
		return HomeLinks{}, errors.New(`studio: home document missing link "categories"`)
	}
	artworks, ok := home.Links["artworks"]
	if !ok {
		return HomeLinks{}, errors.New(`studio: home document missing link "artworks"`)
	}
	artwork, ok := home.Links["artwork"]
	if !ok {
		return HomeLinks{}, errors.New(`studio: home document missing link "artwork"`)
	}
	createArtwork, ok := home.Links["create-artwork"]
	if !ok {
		return HomeLinks{}, errors.New(`studio: home document missing link "create-artwork"`)
	}
	return HomeLinks{
		Me:            me,
		Categories:    categories,
		Artworks:      artworks,
		Artwork:       artwork,
		CreateArtwork: createArtwork,
	}, nil
}
