package studio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func ptr[T any](value T) *T {
	return &value
}

func readArtworksFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/artworks.json")
	if err != nil {
		t.Fatalf("read testdata/artworks.json: %v", err)
	}
	return data
}

func decodeFields(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return fields
}

func artworkJSON(id string, version int) string {
	return fmt.Sprintf(`{
		"id": %q,
		"version": %d,
		"published": false,
		"images": [],
		"sortOrder": 0,
		"tags": {},
		"createdAt": "2026-11-09T10:00:00Z",
		"updatedAt": "2026-11-09T10:00:00Z",
		"createdBy": {"subject": "user_1", "kind": "artist"},
		"updatedBy": {"subject": "user_1", "kind": "artist"},
		"links": {
			"self": {"href": "/artworks/%s", "rel": "self", "httpMethod": "GET"},
			"update": {"href": "/artworks/%s", "rel": "update", "httpMethod": "PATCH"}
		}
	}`, id, version, id, id)
}

func TestArtworkCollection_GoldenDocument(t *testing.T) {
	t.Parallel()
	var collection ArtworkCollection
	if err := json.Unmarshal(readArtworksFixture(t), &collection); err != nil {
		t.Fatalf("decode testdata/artworks.json: %v", err)
	}
	firstID := "01ARZ3NDEKTSV4RRFFQ69G5F02"
	secondID := "01ARZ3NDEKTSV4RRFFQ69G5F03"
	want := []Artwork{
		{
			ID:        firstID,
			Version:   3,
			Published: true,
			Slug:      &LocalizedText{PTBR: "the-path", EN: "the-path"},
			Title:     &LocalizedText{PTBR: "The Path", EN: "The Path"},
			Year:      ptr(2023),
			Medium:    &LocalizedText{PTBR: "Óleo sobre tela", EN: "Oil on canvas"},
			Dimensions: &Dimensions{
				WidthCm:  40,
				HeightCm: 60,
				DepthCm:  ptr(2.5),
			},
			CategoryID: "animals",
			Description: &LocalizedText{
				PTBR: "A noite não tem pressa. O que atravessa a tela também não.",
				EN:   "The night is in no hurry. Neither is what crosses the canvas.",
			},
			Images: []ArtworkImage{
				{
					MediaID:   "01ARZ3NDEKTSV4RRFFQ69G5M01",
					Role:      ImageRoleMain,
					SortOrder: 0,
					AltText: &LocalizedText{
						PTBR: "Um elefante caminha sob a lua, entre a poeira e a luz.",
						EN:   "An elephant walks under the moon, through dust and light.",
					},
				},
				{
					MediaID:   "01ARZ3NDEKTSV4RRFFQ69G5M02",
					Role:      ImageRoleDetail,
					SortOrder: 1,
					AltText: &LocalizedText{
						PTBR: "Detalhe da pincelada na poeira levantada pelo elefante.",
						EN:   "Detail of the brushwork in the dust raised by the elephant.",
					},
				},
			},
			SortOrder: 1,
			Tags: map[string]string{
				"availability": "commission-only",
				"priority":     "medium",
			},
			Exhibition: &Exhibition{
				Label: "Louvre 2023",
				Text: LocalizedText{
					PTBR: "Obra exibida no Carrousel du Louvre, em outubro de 2023.",
					EN:   "Shown at the Carrousel du Louvre in October 2023.",
				},
			},
			Note:      &LocalizedText{PTBR: "Série Noite", EN: "Night series"},
			CreatedAt: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 11, 8, 9, 30, 0, 0, time.UTC),
			CreatedBy: Actor{Subject: "user_1", Kind: KindArtist},
			UpdatedBy: Actor{Subject: "user_1", Kind: KindArtist},
			Links: ArtworkLinks{
				Self:   Link{Href: "/artworks/" + firstID, Rel: "self", HTTPMethod: http.MethodGet},
				Update: Link{Href: "/artworks/" + firstID, Rel: "update", HTTPMethod: http.MethodPatch},
			},
		},
		{
			ID:        secondID,
			Version:   1,
			Published: false,
			Images: []ArtworkImage{
				{
					MediaID:   "01ARZ3NDEKTSV4RRFFQ69G5M03",
					Role:      ImageRoleMain,
					SortOrder: 0,
				},
			},
			SortOrder: 0,
			Tags:      map[string]string{},
			CreatedAt: time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC),
			CreatedBy: Actor{Subject: "user_1", Kind: KindArtist},
			UpdatedBy: Actor{Subject: "user_1", Kind: KindArtist},
			Links: ArtworkLinks{
				Self:   Link{Href: "/artworks/" + secondID, Rel: "self", HTTPMethod: http.MethodGet},
				Update: Link{Href: "/artworks/" + secondID, Rel: "update", HTTPMethod: http.MethodPatch},
			},
		},
	}
	if !reflect.DeepEqual(collection.Artworks, want) {
		t.Errorf("Artworks = %+v, want %+v", collection.Artworks, want)
	}
	if got, want := collection.NextCursor, "eyJwayI6IkFSVFdPUksjMDEifQ"; got != want {
		t.Errorf("NextCursor = %q, want %q", got, want)
	}
	if got, want := collection.Links.Self.Href, "/artworks?limit=2"; got != want {
		t.Errorf("Links.Self.Href = %q, want %q", got, want)
	}
	if collection.Links.Next == nil {
		t.Fatal("Links.Next = nil, want a next link")
	}
	if got, want := collection.Links.Next.Href, "/artworks?cursor=eyJwayI6IkFSVFdPUksjMDEifQ&limit=2"; got != want {
		t.Errorf("Links.Next.Href = %q, want %q", got, want)
	}
	if got, want := collection.Links.Next.Rel, "next"; got != want {
		t.Errorf("Links.Next.Rel = %q, want %q", got, want)
	}
}

func TestArtworkImage_AltTextRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		wire     string
		wantAlt  *LocalizedText
		wantText string
	}{
		{
			name: "a draft image without alt text",
			wire: `{"mediaId":"01ARZ3NDEKTSV4RRFFQ69G5M03","role":"main","sortOrder":0}`,
		},
		{
			name:     "an image with alt text",
			wire:     `{"mediaId":"01ARZ3NDEKTSV4RRFFQ69G5M01","role":"main","sortOrder":0,"altText":{"ptBR":"Um elefante caminha sob a lua.","en":"An elephant walks under the moon."}}`,
			wantAlt:  &LocalizedText{PTBR: "Um elefante caminha sob a lua.", EN: "An elephant walks under the moon."},
			wantText: `{"ptBR":"Um elefante caminha sob a lua.","en":"An elephant walks under the moon."}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var image ArtworkImage
			if err := json.Unmarshal([]byte(tt.wire), &image); err != nil {
				t.Fatalf("Unmarshal() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(image.AltText, tt.wantAlt) {
				t.Errorf("AltText = %+v, want %+v", image.AltText, tt.wantAlt)
			}

			encoded, err := json.Marshal(image)
			if err != nil {
				t.Fatalf("Marshal() error = %v, want nil", err)
			}
			fields := decodeFields(t, encoded)
			got, ok := fields["altText"]
			if tt.wantText == "" {
				if ok {
					t.Errorf("body[\"altText\"] = %s, want the field absent", got)
				}
				return
			}
			if !ok {
				t.Fatalf("body[\"altText\"] is absent, want %s", tt.wantText)
			}
			if string(got) != tt.wantText {
				t.Errorf("body[\"altText\"] = %s, want %s", got, tt.wantText)
			}
		})
	}
}

func TestGetArtwork_ExpandsTemplatedLink(t *testing.T) {
	t.Parallel()
	const id = "01ARZ3NDEKTSV4RRFFQ69G5F02"
	paths := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		writeJSON(t, w, http.StatusOK, artworkJSON(id, 1))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	artwork, err := GetArtwork(context.Background(), client, Link{
		Href:       "/artworks/{id}",
		Rel:        "artwork",
		HTTPMethod: http.MethodGet,
		Templated:  true,
	}, id)
	if err != nil {
		t.Fatalf("GetArtwork() error = %v, want nil", err)
	}
	if got, want := <-paths, "/artworks/"+id; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if got, want := artwork.ID, id; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := artwork.Links.Update.HTTPMethod, http.MethodPatch; got != want {
		t.Errorf("Links.Update.HTTPMethod = %q, want %q", got, want)
	}
}

func TestGetArtwork_RejectsEmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://api.example.com", staticTokenSource("access-token"))
	_, err := GetArtwork(context.Background(), client, Link{
		Href:       "/artworks/{id}",
		Rel:        "artwork",
		HTTPMethod: http.MethodGet,
		Templated:  true,
	}, "")
	if err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("GetArtwork() error = %v, want an error naming the missing id", err)
	}
}

func TestListArtworks_RejectsNegativeLimit(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://api.example.com", staticTokenSource("access-token"))
	_, err := ListArtworks(context.Background(), client, Link{
		Href:       "/artworks",
		Rel:        "artworks",
		HTTPMethod: http.MethodGet,
	}, ArtworkFilter{Limit: -1})
	if err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("ListArtworks() error = %v, want an error naming the negative limit", err)
	}
}

func TestListArtworks_AppliesFilter(t *testing.T) {
	t.Parallel()
	fixture := readArtworksFixture(t)
	tests := []struct {
		name   string
		filter ArtworkFilter
		want   map[string]string
	}{
		{
			name:   "sends no query without a filter",
			filter: ArtworkFilter{},
			want:   map[string]string{},
		},
		{
			name:   "sends published false and a category",
			filter: ArtworkFilter{Published: ptr(false), CategoryID: "animals"},
			want:   map[string]string{"published": "false", "categoryId": "animals"},
		},
		{
			name:   "sends the limit and cursor",
			filter: ArtworkFilter{Published: ptr(true), Limit: 2, Cursor: "eyJwayI6IkFSVFdPUksjMDEifQ"},
			want:   map[string]string{"published": "true", "limit": "2", "cursor": "eyJwayI6IkFSVFdPUksjMDEifQ"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			queries := make(chan url.Values, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				queries <- r.URL.Query()
				writeJSON(t, w, http.StatusOK, string(fixture))
			}))
			t.Cleanup(server.Close)

			client := newTestClient(t, server.URL, staticTokenSource("access-token"))
			collection, err := ListArtworks(context.Background(), client, Link{
				Href:       "/artworks",
				Rel:        "artworks",
				HTTPMethod: http.MethodGet,
			}, tt.filter)
			if err != nil {
				t.Fatalf("ListArtworks() error = %v, want nil", err)
			}

			query := <-queries
			if got, want := len(query), len(tt.want); got != want {
				t.Errorf("len(query) = %d, want %d: %v", got, want, query)
			}
			for name, want := range tt.want {
				if got := query.Get(name); got != want {
					t.Errorf("query[%q] = %q, want %q", name, got, want)
				}
			}
			if len(collection.Artworks) == 0 {
				t.Fatal("len(Artworks) = 0, want the fixture's artworks")
			}
			if got, want := collection.Artworks[0].ID, "01ARZ3NDEKTSV4RRFFQ69G5F02"; got != want {
				t.Errorf("Artworks[0].ID = %q, want %q", got, want)
			}
			if got, want := collection.NextCursor, "eyJwayI6IkFSVFdPUksjMDEifQ"; got != want {
				t.Errorf("NextCursor = %q, want %q", got, want)
			}
		})
	}
}

type observedRequest struct {
	method         string
	path           string
	contentType    string
	idempotencyKey string
	authorization  string
	body           map[string]json.RawMessage
}

func TestCreateArtwork_SendsDraftAndHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		idempotencyKey string
	}{
		{name: "without an idempotency key"},
		{name: "with an idempotency key", idempotencyKey: "create-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const id = "01ARZ3NDEKTSV4RRFFQ69G5F04"
			requests := make(chan observedRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request body: %v", err)
				}
				requests <- observedRequest{
					method:         r.Method,
					path:           r.URL.Path,
					contentType:    r.Header.Get("Content-Type"),
					idempotencyKey: r.Header.Get("Idempotency-Key"),
					authorization:  r.Header.Get("Authorization"),
					body:           body,
				}
				writeJSON(t, w, http.StatusCreated, artworkJSON(id, 1))
			}))
			t.Cleanup(server.Close)

			client := newTestClient(t, server.URL, staticTokenSource("access-token"))
			create := ArtworkCreate{
				Title:      &LocalizedText{PTBR: "O Caminho", EN: "The Path"},
				Year:       ptr(2023),
				CategoryID: "animals",
			}
			artwork, err := CreateArtwork(context.Background(), client, Link{
				Href:       "/artworks",
				Rel:        "create-artwork",
				HTTPMethod: http.MethodPost,
			}, create, tt.idempotencyKey)
			if err != nil {
				t.Fatalf("CreateArtwork() error = %v, want nil", err)
			}

			request := <-requests
			if got, want := request.method, http.MethodPost; got != want {
				t.Errorf("method = %q, want %q", got, want)
			}
			if got, want := request.path, "/artworks"; got != want {
				t.Errorf("path = %q, want %q", got, want)
			}
			if got, want := request.contentType, "application/json"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			if got, want := request.idempotencyKey, tt.idempotencyKey; got != want {
				t.Errorf("Idempotency-Key = %q, want %q", got, want)
			}
			if got, want := request.authorization, "Bearer access-token"; got != want {
				t.Errorf("Authorization = %q, want %q", got, want)
			}

			wantBody := map[string]string{
				"title":      `{"ptBR":"O Caminho","en":"The Path"}`,
				"year":       "2023",
				"categoryId": `"animals"`,
			}
			for name, want := range wantBody {
				got, ok := request.body[name]
				if !ok {
					t.Errorf("body[%q] is absent, want %s", name, want)
					continue
				}
				if string(got) != want {
					t.Errorf("body[%q] = %s, want %s", name, got, want)
				}
			}
			for _, name := range []string{
				"id", "version", "published", "slug", "medium", "dimensions", "description",
				"images", "sortOrder", "tags", "exhibition", "note", "createdAt", "updatedAt",
				"createdBy", "updatedBy", "links",
			} {
				if _, ok := request.body[name]; ok {
					t.Errorf("body[%q] is present, want a server-owned field absent", name)
				}
			}
			if got, want := artwork.ID, id; got != want {
				t.Errorf("ID = %q, want %q", got, want)
			}
			if got, want := artwork.Version, 1; got != want {
				t.Errorf("Version = %d, want %d", got, want)
			}
		})
	}
}

func TestCreateArtwork_DoesNotRetry(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		writeJSON(t, w, http.StatusServiceUnavailable, `{"message":"Service Unavailable"}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	_, err := CreateArtwork(context.Background(), client, Link{
		Href:       "/artworks",
		Rel:        "create-artwork",
		HTTPMethod: http.MethodPost,
	}, ArtworkCreate{}, "create-1")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CreateArtwork() error = %v, want *APIError", err)
	}
	if got, want := apiErr.Status, http.StatusServiceUnavailable; got != want {
		t.Errorf("Status = %d, want %d", got, want)
	}
	if got, want := attempts.Load(), int32(1); got != want {
		t.Errorf("attempts = %d, want %d", got, want)
	}
}

func TestUpdateArtwork_SendsPatch(t *testing.T) {
	t.Parallel()
	const id = "01ARZ3NDEKTSV4RRFFQ69G5F02"
	requests := make(chan observedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		requests <- observedRequest{
			method:         r.Method,
			path:           r.URL.Path,
			contentType:    r.Header.Get("Content-Type"),
			idempotencyKey: r.Header.Get("Idempotency-Key"),
			authorization:  r.Header.Get("Authorization"),
			body:           body,
		}
		writeJSON(t, w, http.StatusOK, artworkJSON(id, 4))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	patch := ArtworkPatch{
		Version:    3,
		Published:  Some(true),
		Title:      Some(LocalizedText{PTBR: "O Caminho", EN: "The Path"}),
		Year:       Null[int](),
		CategoryID: Null[string](),
	}
	artwork, err := UpdateArtwork(context.Background(), client, Link{
		Href:       "/artworks/" + id,
		Rel:        "update",
		HTTPMethod: http.MethodPatch,
	}, patch, "")
	if err != nil {
		t.Fatalf("UpdateArtwork() error = %v, want nil", err)
	}

	request := <-requests
	if got, want := request.method, http.MethodPatch; got != want {
		t.Errorf("method = %q, want %q", got, want)
	}
	if got, want := request.path, "/artworks/"+id; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if got, want := request.contentType, "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := request.idempotencyKey, ""; got != want {
		t.Errorf("Idempotency-Key = %q, want %q", got, want)
	}
	wantBody := map[string]string{
		"version":    "3",
		"published":  "true",
		"title":      `{"ptBR":"O Caminho","en":"The Path"}`,
		"year":       "null",
		"categoryId": "null",
	}
	for name, want := range wantBody {
		got, ok := request.body[name]
		if !ok {
			t.Errorf("body[%q] is absent, want %s", name, want)
			continue
		}
		if string(got) != want {
			t.Errorf("body[%q] = %s, want %s", name, got, want)
		}
	}
	for _, name := range []string{"slug", "medium", "dimensions", "description", "images", "sortOrder", "tags", "exhibition", "note"} {
		if _, ok := request.body[name]; ok {
			t.Errorf("body[%q] is present, want the unset field absent", name)
		}
	}
	if got, want := artwork.Version, 4; got != want {
		t.Errorf("Version = %d, want %d", got, want)
	}
}

func TestUpdateArtwork_RejectsNonMutationLink(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://api.example.com", staticTokenSource("access-token"))
	_, err := UpdateArtwork(context.Background(), client, Link{
		Href:       "/artworks/01ARZ3NDEKTSV4RRFFQ69G5F02",
		Rel:        "update",
		HTTPMethod: http.MethodGet,
	}, ArtworkPatch{Version: 1}, "")
	if err == nil || !strings.Contains(err.Error(), "not a mutation") {
		t.Fatalf("UpdateArtwork() error = %v, want an error naming a non-mutation link", err)
	}
}

func TestArtworkPatch_DistinguishesAbsentFromNull(t *testing.T) {
	t.Parallel()
	t.Run("encodes unset fields away and null fields as null", func(t *testing.T) {
		t.Parallel()
		patch := ArtworkPatch{
			Version:    4,
			Published:  Some(true),
			Title:      Some(LocalizedText{PTBR: "O Caminho", EN: "The Path"}),
			Images:     Some([]ArtworkImage{{MediaID: "01ARZ3NDEKTSV4RRFFQ69G5M01", Role: ImageRoleMain}}),
			SortOrder:  Some(0),
			Tags:       Some(map[string]string{}),
			Slug:       Null[LocalizedText](),
			Year:       Null[int](),
			Dimensions: Null[Dimensions](),
			CategoryID: Null[string](),
			Exhibition: Null[Exhibition](),
			Note:       Null[LocalizedText](),
		}
		encoded, err := json.Marshal(patch)
		if err != nil {
			t.Fatalf("Marshal() error = %v, want nil", err)
		}
		fields := decodeFields(t, encoded)

		for _, name := range []string{"slug", "year", "dimensions", "categoryId", "exhibition", "note"} {
			got, ok := fields[name]
			if !ok {
				t.Errorf("body[%q] is absent, want null", name)
				continue
			}
			if string(got) != "null" {
				t.Errorf("body[%q] = %s, want null", name, got)
			}
		}
		want := map[string]string{
			"version":   "4",
			"published": "true",
			"title":     `{"ptBR":"O Caminho","en":"The Path"}`,
			"sortOrder": "0",
			"tags":      "{}",
		}
		for name, want := range want {
			got, ok := fields[name]
			if !ok {
				t.Errorf("body[%q] is absent, want %s", name, want)
				continue
			}
			if string(got) != want {
				t.Errorf("body[%q] = %s, want %s", name, got, want)
			}
		}
		rawImages, ok := fields["images"]
		if !ok {
			t.Fatal("body[\"images\"] is absent, want the replacement list")
		}
		var images []map[string]json.RawMessage
		if err := json.Unmarshal(rawImages, &images); err != nil {
			t.Fatalf("decode body[\"images\"]: %v", err)
		}
		if len(images) != 1 {
			t.Fatalf("len(body[\"images\"]) = %d, want 1", len(images))
		}
		if _, ok := images[0]["altText"]; ok {
			t.Error("body[\"images\"][0][\"altText\"] is present, want it absent on an image without alt text")
		}
		for _, name := range []string{"medium", "description"} {
			if _, ok := fields[name]; ok {
				t.Errorf("body[%q] is present, want the unset field absent", name)
			}
		}
	})

	t.Run("decodes absent fields and null fields differently", func(t *testing.T) {
		t.Parallel()
		source := `{"version":2,"title":{"ptBR":"O Caminho","en":"The Path"},"year":null,"note":null}`
		var patch ArtworkPatch
		if err := json.Unmarshal([]byte(source), &patch); err != nil {
			t.Fatalf("Unmarshal() error = %v, want nil", err)
		}
		encoded, err := json.Marshal(patch)
		if err != nil {
			t.Fatalf("Marshal() error = %v, want nil", err)
		}
		fields := decodeFields(t, encoded)

		want := map[string]string{
			"version": "2",
			"title":   `{"ptBR":"O Caminho","en":"The Path"}`,
			"year":    "null",
			"note":    "null",
		}
		for name, want := range want {
			got, ok := fields[name]
			if !ok {
				t.Errorf("body[%q] is absent, want %s", name, want)
				continue
			}
			if string(got) != want {
				t.Errorf("body[%q] = %s, want %s", name, got, want)
			}
		}
		for _, name := range []string{"slug", "medium", "dimensions", "categoryId", "description", "images", "published"} {
			if _, ok := fields[name]; ok {
				t.Errorf("body[%q] is present, want the absent field absent", name)
			}
		}
	})
}
