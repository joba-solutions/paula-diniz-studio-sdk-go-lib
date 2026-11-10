package studio

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// ImageRole is the part an image plays on the artwork page.
type ImageRole string

const (
	// ImageRoleMain is the source image shown on every card and page.
	ImageRoleMain ImageRole = "main"
	// ImageRoleDetail is a close-up of the work.
	ImageRoleDetail ImageRole = "detail"
	// ImageRoleProcess is the work in progress.
	ImageRoleProcess ImageRole = "process"
	// ImageRoleRoom is the work shown in place.
	ImageRoleRoom ImageRole = "room"
)

// Dimensions is the work's size in centimetres, measured without the frame.
type Dimensions struct {
	WidthCm  float64  `json:"widthCm"`
	HeightCm float64  `json:"heightCm"`
	DepthCm  *float64 `json:"depthCm,omitempty"`
}

// ArtworkImage references an uploaded image. Alt text is required to publish and absent on a draft.
type ArtworkImage struct {
	MediaID   string         `json:"mediaId"`
	Role      ImageRole      `json:"role"`
	SortOrder int            `json:"sortOrder"`
	AltText   *LocalizedText `json:"altText,omitempty"`
}

// Exhibition is where the work was shown: a label and its story in both languages.
type Exhibition struct {
	Label string        `json:"label"`
	Text  LocalizedText `json:"text"`
}

// Actor is who wrote a record: the verified Clerk subject and its kind.
type Actor struct {
	Subject string `json:"subject"`
	Kind    Kind   `json:"kind"`
}

// ArtworkLinks are the actions the artwork resource advertises.
type ArtworkLinks struct {
	Self   Link `json:"self"`
	Update Link `json:"update"`
}

// Artwork is one painting with its metadata, images, tags, and publish state.
type Artwork struct {
	ID          string            `json:"id"`
	Version     int               `json:"version"`
	Published   bool              `json:"published"`
	Slug        *LocalizedText    `json:"slug,omitempty"`
	Title       *LocalizedText    `json:"title,omitempty"`
	Year        *int              `json:"year,omitempty"`
	Medium      *LocalizedText    `json:"medium,omitempty"`
	Dimensions  *Dimensions       `json:"dimensions,omitempty"`
	CategoryID  string            `json:"categoryId,omitempty"`
	Description *LocalizedText    `json:"description,omitempty"`
	Images      []ArtworkImage    `json:"images"`
	SortOrder   int               `json:"sortOrder"`
	Tags        map[string]string `json:"tags"`
	Exhibition  *Exhibition       `json:"exhibition,omitempty"`
	Note        *LocalizedText    `json:"note,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	CreatedBy   Actor             `json:"createdBy"`
	UpdatedBy   Actor             `json:"updatedBy"`
	Links       ArtworkLinks      `json:"links"`
}

// ArtworkCreate is the optional metadata of a new draft. The server owns the id, version,
// timestamps, actor, publish state, and links.
type ArtworkCreate struct {
	Slug        *LocalizedText    `json:"slug,omitempty"`
	Title       *LocalizedText    `json:"title,omitempty"`
	Year        *int              `json:"year,omitempty"`
	Medium      *LocalizedText    `json:"medium,omitempty"`
	Dimensions  *Dimensions       `json:"dimensions,omitempty"`
	CategoryID  string            `json:"categoryId,omitempty"`
	Description *LocalizedText    `json:"description,omitempty"`
	Images      []ArtworkImage    `json:"images,omitempty"`
	SortOrder   int               `json:"sortOrder,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Exhibition  *Exhibition       `json:"exhibition,omitempty"`
	Note        *LocalizedText    `json:"note,omitempty"`
}

// ArtworkPatch is a partial update. A field left unset is absent from the request, Some sets it,
// and Null sends an explicit null. The API clears year, dimensions, categoryId, exhibition, note,
// and slug with null; version is required.
type ArtworkPatch struct {
	Version     int                         `json:"version"`
	Published   Optional[bool]              `json:"published,omitzero"`
	Slug        Optional[LocalizedText]     `json:"slug,omitzero"`
	Title       Optional[LocalizedText]     `json:"title,omitzero"`
	Year        Optional[int]               `json:"year,omitzero"`
	Medium      Optional[LocalizedText]     `json:"medium,omitzero"`
	Dimensions  Optional[Dimensions]        `json:"dimensions,omitzero"`
	CategoryID  Optional[string]            `json:"categoryId,omitzero"`
	Description Optional[LocalizedText]     `json:"description,omitzero"`
	Images      Optional[[]ArtworkImage]    `json:"images,omitzero"`
	SortOrder   Optional[int]               `json:"sortOrder,omitzero"`
	Tags        Optional[map[string]string] `json:"tags,omitzero"`
	Exhibition  Optional[Exhibition]        `json:"exhibition,omitzero"`
	Note        Optional[LocalizedText]     `json:"note,omitzero"`
}

// ArtworkFilter narrows the artwork list. A nil Published, an empty CategoryID or Cursor, and a
// zero Limit are not sent.
type ArtworkFilter struct {
	Published  *bool
	CategoryID string
	Limit      int
	Cursor     string
}

// ArtworkCollectionLinks are the links on one artwork page.
type ArtworkCollectionLinks struct {
	Self Link  `json:"self"`
	Next *Link `json:"next,omitempty"`
}

// ArtworkCollection is one page of artworks in editorial order.
type ArtworkCollection struct {
	Artworks   []Artwork              `json:"artworks"`
	NextCursor string                 `json:"nextCursor,omitempty"`
	Links      ArtworkCollectionLinks `json:"links"`
}

// GetArtwork expands the templated artwork link for id and returns the artwork.
func GetArtwork(ctx context.Context, client *Client, artworkLink Link, id string) (Artwork, error) {
	if id == "" {
		return Artwork{}, errors.New("studio: artwork id is required")
	}
	return Get[Artwork](ctx, client, Expand(artworkLink, map[string]string{"id": id}))
}

// ListArtworks follows the artworks link with the filter and returns one page. The caller follows
// collection.Links.Next for the next page.
func ListArtworks(ctx context.Context, client *Client, artworksLink Link, filter ArtworkFilter) (ArtworkCollection, error) {
	if filter.Limit < 0 {
		return ArtworkCollection{}, fmt.Errorf("studio: artwork limit must not be negative, got %d", filter.Limit)
	}
	query := map[string]string{
		"categoryId": filter.CategoryID,
		"cursor":     filter.Cursor,
	}
	if filter.Published != nil {
		query["published"] = strconv.FormatBool(*filter.Published)
	}
	if filter.Limit > 0 {
		query["limit"] = strconv.Itoa(filter.Limit)
	}
	return Get[ArtworkCollection](ctx, client, WithQuery(artworksLink, query))
}

// CreateArtwork posts the draft to the create link and returns the created artwork. The
// Idempotency-Key header is sent when idempotencyKey is not empty.
func CreateArtwork(ctx context.Context, client *Client, createLink Link, create ArtworkCreate, idempotencyKey string) (Artwork, error) {
	return mutate[Artwork](ctx, client, createLink, create, idempotencyKey)
}

// UpdateArtwork patches the artwork at the update link and returns the updated artwork. The
// Idempotency-Key header is sent when idempotencyKey is not empty.
func UpdateArtwork(ctx context.Context, client *Client, updateLink Link, patch ArtworkPatch, idempotencyKey string) (Artwork, error) {
	return mutate[Artwork](ctx, client, updateLink, patch, idempotencyKey)
}
