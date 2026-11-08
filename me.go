package studio

// Kind is the kind of caller the Studio admits.
type Kind string

// KindArtist is the artist herself, using the CLI or the iOS app.
const KindArtist Kind = "artist"

// MeLinks are the links a caller can follow from its identity.
type MeLinks struct {
	Self Link `json:"self"`
}

// Me is the verified caller, as GET /me returns it.
type Me struct {
	Subject string  `json:"subject"`
	Kind    Kind    `json:"kind"`
	Links   MeLinks `json:"links"`
}
