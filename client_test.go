package studio

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type staticTokenSource string

func (s staticTokenSource) Token(context.Context) (string, error) {
	return string(s), nil
}

func newTestClient(t *testing.T, base string, tokens TokenSource) *Client {
	t.Helper()
	client, err := NewClient(ClientOptions{Base: base, TokenSource: tokens})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	client.retryDelay = func(int) time.Duration { return 0 }
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func followLink(t *testing.T, client *Client, link Link) error {
	t.Helper()
	response, err := client.Follow(context.Background(), link)
	if response != nil {
		closeErr := response.Body.Close()
		if closeErr != nil {
			t.Errorf("close response body: %v", closeErr)
		}
	}
	return err
}

func TestNewClient_RejectsInvalidOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		options ClientOptions
		wantErr string
	}{
		{
			name:    "empty base",
			options: ClientOptions{TokenSource: staticTokenSource("access-token")},
			wantErr: "base URL is required",
		},
		{
			name:    "relative base",
			options: ClientOptions{Base: "api.example.com", TokenSource: staticTokenSource("access-token")},
			wantErr: "absolute HTTP or HTTPS",
		},
		{
			name:    "unsupported scheme",
			options: ClientOptions{Base: "ftp://api.example.com", TokenSource: staticTokenSource("access-token")},
			wantErr: "absolute HTTP or HTTPS",
		},
		{
			name:    "missing token source",
			options: ClientOptions{Base: "https://api.example.com"},
			wantErr: "token source is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(tt.options)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewClient() error = %v, want error containing %q", err, tt.wantErr)
			}
			if client != nil {
				t.Errorf("NewClient() client = %v, want nil", client)
			}
		})
	}
}

func TestNewClient_HTTPClient(t *testing.T) {
	t.Parallel()
	t.Run("defaults to a client with a timeout", func(t *testing.T) {
		t.Parallel()
		client, err := NewClient(ClientOptions{
			Base:        "https://api.example.com",
			TokenSource: staticTokenSource("access-token"),
		})
		if err != nil {
			t.Fatalf("NewClient() error = %v, want nil", err)
		}
		if got, want := client.httpClient.Timeout, defaultTimeout; got != want {
			t.Errorf("HTTP client timeout = %v, want %v", got, want)
		}
	})
	t.Run("uses the provided client", func(t *testing.T) {
		t.Parallel()
		provided := &http.Client{Timeout: time.Second}
		client, err := NewClient(ClientOptions{
			Base:        "https://api.example.com",
			TokenSource: staticTokenSource("access-token"),
			HTTPClient:  provided,
		})
		if err != nil {
			t.Fatalf("NewClient() error = %v, want nil", err)
		}
		if client.httpClient != provided {
			t.Errorf("HTTP client = %v, want the provided client", client.httpClient)
		}
	})
}

func TestFollow_SetsBearerToken(t *testing.T) {
	t.Parallel()
	type observedRequest struct {
		path          string
		authorization string
	}
	requests := make(chan observedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- observedRequest{path: r.URL.Path, authorization: r.Header.Get("Authorization")}
		writeJSON(t, w, http.StatusOK, `{"links":{}}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	if _, err := Get[Home](context.Background(), client, HomeLink); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	request := <-requests
	if got, want := request.path, "/"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if got, want := request.authorization, "Bearer access-token"; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestGet_ReadsHomeThenMe(t *testing.T) {
	t.Parallel()
	fixture, err := os.ReadFile("testdata/home.json")
	if err != nil {
		t.Fatalf("read testdata/home.json: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			writeJSON(t, w, http.StatusOK, string(fixture))
		case "/me":
			writeJSON(t, w, http.StatusOK, `{"subject":"user_1","kind":"artist","links":{"self":{"href":"/me","rel":"self","httpMethod":"GET"}}}`)
		default:
			writeJSON(t, w, http.StatusNotFound, `{"message":"Not Found"}`)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	document, err := Get[Home](context.Background(), client, HomeLink)
	if err != nil {
		t.Fatalf("Get[Home]() error = %v, want nil", err)
	}
	links, err := ResolveHome(document)
	if err != nil {
		t.Fatalf("ResolveHome() error = %v, want nil", err)
	}
	me, err := Get[Me](context.Background(), client, links.Me)
	if err != nil {
		t.Fatalf("Get[Me]() error = %v, want nil", err)
	}

	if got, want := me.Subject, "user_1"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
	if got, want := me.Kind, KindArtist; got != want {
		t.Errorf("Kind = %q, want %q", got, want)
	}
	self := Link{Href: "/me", Rel: "self", HTTPMethod: "GET"}
	if me.Links.Self != self {
		t.Errorf("Links.Self = %+v, want %+v", me.Links.Self, self)
	}
}

func TestFollow_RetriesTransientGetFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
	}{
		{name: "too many requests", status: http.StatusTooManyRequests},
		{name: "internal server error", status: http.StatusInternalServerError},
		{name: "bad gateway", status: http.StatusBadGateway},
		{name: "service unavailable", status: http.StatusServiceUnavailable},
		{name: "gateway timeout", status: http.StatusGatewayTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if attempts.Add(1) == 1 {
					writeJSON(t, w, tt.status, `{"message":"transient"}`)
					return
				}
				writeJSON(t, w, http.StatusOK, `{"subject":"user_1","kind":"artist","links":{"self":{"href":"/me","rel":"self","httpMethod":"GET"}}}`)
			}))
			t.Cleanup(server.Close)

			client := newTestClient(t, server.URL, staticTokenSource("access-token"))
			me, err := Get[Me](context.Background(), client, Link{Href: "/me", Rel: "me", HTTPMethod: http.MethodGet})
			if err != nil {
				t.Fatalf("Get() error = %v, want nil", err)
			}
			if got, want := me.Subject, "user_1"; got != want {
				t.Errorf("Subject = %q, want %q", got, want)
			}
			if got, want := attempts.Load(), int32(2); got != want {
				t.Errorf("attempts = %d, want %d", got, want)
			}
		})
	}
}

func TestFollow_DoesNotRetryNonTransientStatuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
	}{
		{name: "bad request", status: http.StatusBadRequest},
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "forbidden", status: http.StatusForbidden},
		{name: "not found", status: http.StatusNotFound},
		{name: "conflict", status: http.StatusConflict},
		{name: "unprocessable entity", status: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts.Add(1)
				writeJSON(t, w, tt.status, `{"message":"failure"}`)
			}))
			t.Cleanup(server.Close)

			client := newTestClient(t, server.URL, staticTokenSource("access-token"))
			_, err := Get[Me](context.Background(), client, Link{Href: "/me", Rel: "me", HTTPMethod: http.MethodGet})

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("Get() error = %v, want *APIError", err)
			}
			if got, want := apiErr.Status, tt.status; got != want {
				t.Errorf("Status = %d, want %d", got, want)
			}
			if got, want := attempts.Load(), int32(1); got != want {
				t.Errorf("attempts = %d, want %d", got, want)
			}
		})
	}
}

func TestFollow_DoesNotRetryMutations(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		writeJSON(t, w, http.StatusServiceUnavailable, `{"message":"Service Unavailable"}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	err := followLink(t, client, Link{Href: "/media", Rel: "media", HTTPMethod: http.MethodPost})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Follow() error = %v, want *APIError", err)
	}
	if got, want := apiErr.Status, http.StatusServiceUnavailable; got != want {
		t.Errorf("Status = %d, want %d", got, want)
	}
	if got, want := attempts.Load(), int32(1); got != want {
		t.Errorf("attempts = %d, want %d", got, want)
	}
}

func TestFollow_ReturnsAPIErrorWithBodyAndAuthenticateHeader(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeJSON(t, w, http.StatusUnauthorized, `{"message":"Unauthorized","errors":{"sub":"missing"}}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	err := followLink(t, client, Link{Href: "/me", Rel: "me", HTTPMethod: http.MethodGet})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Follow() error = %v, want *APIError", err)
	}
	if got, want := apiErr.Status, http.StatusUnauthorized; got != want {
		t.Errorf("Status = %d, want %d", got, want)
	}
	if got, want := apiErr.Body.Message, "Unauthorized"; got != want {
		t.Errorf("Body.Message = %q, want %q", got, want)
	}
	if got, want := apiErr.Body.Errors["sub"], "missing"; got != want {
		t.Errorf(`Body.Errors["sub"] = %q, want %q`, got, want)
	}
	if got, want := apiErr.WWWAuthenticate, `Bearer error="invalid_token"`; got != want {
		t.Errorf("WWWAuthenticate = %q, want %q", got, want)
	}
	if got, want := err.Error(), "studio: HTTP 401: Unauthorized"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestFollow_StopsAfterBoundedRetries(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		writeJSON(t, w, http.StatusServiceUnavailable, `{"message":"Service Unavailable"}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, staticTokenSource("access-token"))
	err := followLink(t, client, Link{Href: "/me", Rel: "me", HTTPMethod: http.MethodGet})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Follow() error = %v, want *APIError", err)
	}
	if got, want := attempts.Load(), int32(maxAttempts); got != want {
		t.Errorf("attempts = %d, want %d", got, want)
	}
}

func TestFollow_RejectsLinkWithoutMethod(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://api.example.com", staticTokenSource("access-token"))
	err := followLink(t, client, Link{Href: "/me", Rel: "me"})
	if err == nil || !strings.Contains(err.Error(), "no HTTP method") {
		t.Fatalf("Follow() error = %v, want error naming the missing HTTP method", err)
	}
}

func TestFollow_RejectsLinkOutsideBase(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://api.example.com", staticTokenSource("access-token"))
	err := followLink(t, client, Link{Href: "https://evil.example.com/me", Rel: "me", HTTPMethod: http.MethodGet})
	if err == nil || !strings.Contains(err.Error(), "outside the API base") {
		t.Fatalf("Follow() error = %v, want error naming the link outside the API base", err)
	}
}
