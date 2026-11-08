package studio

// Link is a hypermedia link as the API sends it in a document's links.
type Link struct {
	Href       string `json:"href"`
	Rel        string `json:"rel"`
	HTTPMethod string `json:"httpMethod"`
	Templated  bool   `json:"templated,omitempty"`
}
