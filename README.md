# paula-diniz-studio-sdk-go-lib

Go client for the Paula Diniz Studio API.

Module `github.com/joba-solutions/paula-diniz-studio-sdk-go-lib`, package `studio`.

## Install

```bash
go get github.com/joba-solutions/paula-diniz-studio-sdk-go-lib
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

`Base` is the API origin. Links carry only paths, and the client joins them to `Base`.

`Client.Follow` returns the response for the caller to read and close. `Get[T]` follows the link
and decodes its JSON body into `T`.

GET requests retry `429`, `500`, `502`, `503` and `504` up to three attempts with exponential
backoff. Mutations never retry.

A non-2xx response returns `*studio.APIError` with the status, the parsed `{message, errors}` body,
and the edge's `www-authenticate` reason. The token is read from `TokenSource` for every request and
is never logged.
