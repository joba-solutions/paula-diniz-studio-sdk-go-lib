package studio

import (
	"net/url"
	"strings"
)

// Link is a hypermedia link as the API sends it in a document's links.
type Link struct {
	Href       string `json:"href"`
	Rel        string `json:"rel"`
	HTTPMethod string `json:"httpMethod"`
	Templated  bool   `json:"templated,omitempty"`
}

// Expand replaces the link's {placeholders} with the values and returns a concrete link.
func Expand(link Link, params map[string]string) Link {
	for name, value := range params {
		link.Href = strings.ReplaceAll(link.Href, "{"+name+"}", url.PathEscape(value))
	}
	link.Templated = false
	return link
}

// WithQuery appends the query parameters to the link, skipping empty values.
func WithQuery(link Link, query map[string]string) Link {
	values := url.Values{}
	for name, value := range query {
		if value != "" {
			values.Set(name, value)
		}
	}
	encoded := values.Encode()
	if encoded == "" {
		return link
	}
	separator := "?"
	if strings.Contains(link.Href, "?") {
		separator = "&"
	}
	link.Href += separator + encoded
	return link
}
