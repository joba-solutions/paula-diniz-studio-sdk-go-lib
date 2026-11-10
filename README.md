# paula-diniz-studio-sdk-go-lib

Go client for the Paula Diniz Studio API.

Module `github.com/joba-solutions/paula-diniz-studio-sdk-go-lib`, package `studio`.

## Install

```bash
go get github.com/joba-solutions/paula-diniz-studio-sdk-go-lib
```

## Release

Merging to `main` computes the next version from the last tag and the commit messages, then pushes the tag. `#major` and `#minor` in the commit message select the increment; otherwise the patch increases. The tag is the published module, and a tag is never moved.

Consumers pin the tag:

```bash
go get github.com/joba-solutions/paula-diniz-studio-sdk-go-lib@v0.1.0
```

## Use

```go
import (
	"context"

	"github.com/joba-solutions/paula-diniz-studio-sdk-go-lib"
)

// tokenSource supplies a Clerk access token for each request.
type tokenSource struct{}

func (tokenSource) Token(ctx context.Context) (string, error) {
	return "access-token", nil
}

func readMe(ctx context.Context) (studio.Me, error) {
	client, err := studio.NewClient(studio.ClientOptions{
		Base:        "https://api.example.com",
		TokenSource: tokenSource{},
	})
	if err != nil {
		return studio.Me{}, err
	}

	document, err := studio.Get[studio.Home](ctx, client, studio.HomeLink)
	if err != nil {
		return studio.Me{}, err
	}
	links, err := studio.ResolveHome(document)
	if err != nil {
		return studio.Me{}, err
	}
	return studio.Get[studio.Me](ctx, client, links.Me)
}
```

`ResolveHome` requires all five links the home promises: `me`, `categories`, `artworks`,
`artwork` (templated), and `create-artwork`. It fails when the home drops one.

`studio.Categories` is the fixed category list:

```go
categories, err := studio.Get[studio.Categories](ctx, client, links.Categories)
```

Each `studio.Category` carries an `id`, a `slug`, and a `name`. `slug` and `name` are each a
`studio.LocalizedText` with `PTBR` and `EN` values. The five categories come in the API's fixed
order. The collection has a `self` link and no per-item links.

## Artworks

`studio.Artwork` is one work: its localized `slug`, `title`, `medium`, `description`, and `note`,
its `year`, `dimensions`, `categoryId`, `images`, `sortOrder`, `tags`, and `exhibition`, its
`version` and `published` state, its timestamps and actors, and its `self` and `update` links. The
optional fields are pointers and are nil when the API omits them.

`GetArtwork` expands the templated `artwork` link for an id:

```go
artwork, err := studio.GetArtwork(ctx, client, links.Artwork, "01ARZ3NDEKTSV4RRFFQ69G5F02")
```

`ListArtworks` follows the `artworks` link with a filter and returns one page. `Published` is a
pointer so `false` is sent; an empty `CategoryID`, `Cursor`, or zero `Limit` is not sent. Follow
`Links.Next` for the next page.

```go
collection, err := studio.ListArtworks(ctx, client, links.Artworks, studio.ArtworkFilter{
	Published:  &published,
	CategoryID: "animals",
	Limit:      20,
})
// collection.Artworks, collection.NextCursor, collection.Links.Next
```

`CreateArtwork` posts a draft to the `create-artwork` link. `UpdateArtwork` patches the artwork at
the resource's `update` link. Both decode the returned resource and never retry. When the key is
not empty, they send it as `Idempotency-Key`.

```go
created, err := studio.CreateArtwork(ctx, client, links.CreateArtwork, studio.ArtworkCreate{
	Title: &studio.LocalizedText{PTBR: "O Caminho", EN: "The Path"},
}, idempotencyKey)

updated, err := studio.UpdateArtwork(ctx, client, created.Links.Update, studio.ArtworkPatch{
	Version: created.Version,
	Year:    studio.Null[int](),
	Title:   studio.Some(studio.LocalizedText{PTBR: "O Caminho", EN: "The Path"}),
}, idempotencyKey)
```

`studio.ArtworkPatch` is partial: a field left unset is absent from the request,
`studio.Some(value)` sets it, and `studio.Null[T]()` sends an explicit `null`. The API clears
`year`, `dimensions`, `categoryId`, `exhibition`, `note`, and `slug` with `null`. `version` is
always required.

`Expand` replaces a templated link's placeholders and `WithQuery` appends query parameters,
skipping empty values. `GetArtwork` and `ListArtworks` use them.

`Base` is the API origin. Links carry only paths, and the client joins them to `Base`.

`Client.Follow` returns the response for the caller to read and close. `Get[T]` follows the link
and decodes its JSON body into `T`. A mutation sends a JSON body with the `application/json`
content type.

GET requests retry `429`, `500`, `502`, `503` and `504` up to three attempts with exponential
backoff. Mutations never retry.

A non-2xx response returns `*studio.APIError` with the status, the parsed `{message, errors}` body,
and the edge's `www-authenticate` reason. The token is read from `TokenSource` for every request and
is never logged.
