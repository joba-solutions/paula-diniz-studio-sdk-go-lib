package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout   = 30 * time.Second
	maxAttempts      = 3
	baseRetryDelay   = 400 * time.Millisecond
	maxResponseBytes = 1 << 20
	maxErrorBytes    = 1 << 16
)

// TokenSource supplies the bearer token for a request.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// ClientOptions configures a Client. Base is the API origin; links carry only paths.
type ClientOptions struct {
	Base        string
	TokenSource TokenSource
	HTTPClient  *http.Client
}

// Client calls the Studio API with the caller's bearer token.
type Client struct {
	base        *url.URL
	tokenSource TokenSource
	httpClient  *http.Client
	retryDelay  func(attempt int) time.Duration
}

// NewClient validates the options and returns a client.
func NewClient(options ClientOptions) (*Client, error) {
	if options.Base == "" {
		return nil, errors.New("studio: base URL is required")
	}
	base, err := url.Parse(options.Base)
	if err != nil {
		return nil, fmt.Errorf("studio: parse base URL %q: %w", options.Base, err)
	}
	if (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("studio: base URL must be absolute HTTP or HTTPS, got %q", options.Base)
	}
	if options.TokenSource == nil {
		return nil, errors.New("studio: token source is required")
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		base:        base,
		tokenSource: options.TokenSource,
		httpClient:  httpClient,
		retryDelay:  defaultRetryDelay,
	}, nil
}

// Follow sends the link's request with the caller's bearer token and returns the response for the
// caller to read and close. A non-2xx status is an *APIError. GET requests retry transient
// failures; mutations never do.
func (c *Client) Follow(ctx context.Context, link Link) (*http.Response, error) {
	if link.HTTPMethod == "" {
		return nil, fmt.Errorf("studio: link %q has no HTTP method", link.Rel)
	}
	if link.HTTPMethod != http.MethodGet {
		return c.followOnce(ctx, link, nil, "")
	}
	return c.followWithRetry(ctx, link)
}

func (c *Client) followWithRetry(ctx context.Context, link Link) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		response, err := c.followOnce(ctx, link, nil, "")
		if err == nil {
			return response, nil
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || !retryableStatus(apiErr.Status) || attempt == maxAttempts-1 {
			return nil, err
		}
		waitErr := c.wait(ctx, attempt)
		if waitErr != nil {
			return nil, errors.Join(err, waitErr)
		}
	}
}

func (c *Client) followOnce(ctx context.Context, link Link, body any, idempotencyKey string) (*http.Response, error) {
	reference, err := url.Parse(link.Href)
	if err != nil {
		return nil, fmt.Errorf("studio: parse link %q: %w", link.Rel, err)
	}
	requestURL := c.base.ResolveReference(reference)
	if requestURL.Scheme != c.base.Scheme || requestURL.Host != c.base.Host {
		return nil, fmt.Errorf("studio: link %q points outside the API base", link.Rel)
	}
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("studio: encode %q request: %w", link.Rel, err)
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, link.HTTPMethod, requestURL.String(), payload)
	if err != nil {
		return nil, fmt.Errorf("studio: build request for %q: %w", link.Rel, err)
	}
	token, err := c.tokenSource.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("studio: read access token: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("studio: follow %q: %w", link.Rel, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, responseError(response)
	}
	return response, nil
}

func (c *Client) wait(ctx context.Context, attempt int) error {
	timer := time.NewTimer(c.retryDelay(attempt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Get follows the link and decodes its JSON body into T.
func Get[T any](ctx context.Context, client *Client, link Link) (result T, err error) {
	response, err := client.Follow(ctx, link)
	if err != nil {
		return result, err
	}
	defer func() {
		closeErr := response.Body.Close()
		if closeErr != nil {
			err = errors.Join(err, fmt.Errorf("studio: close %q response: %w", link.Rel, closeErr))
		}
	}()
	return decodeBody[T](response.Body, link.Rel)
}

// mutate sends body as the link's JSON request and decodes the returned resource. A mutation
// never retries: a write that already ran must not run twice.
func mutate[T any](ctx context.Context, client *Client, link Link, body any, idempotencyKey string) (result T, err error) {
	if link.HTTPMethod == "" || link.HTTPMethod == http.MethodGet {
		return result, fmt.Errorf("studio: link %q is not a mutation", link.Rel)
	}
	response, err := client.followOnce(ctx, link, body, idempotencyKey)
	if err != nil {
		return result, err
	}
	defer func() {
		closeErr := response.Body.Close()
		if closeErr != nil {
			err = errors.Join(err, fmt.Errorf("studio: close %q response: %w", link.Rel, closeErr))
		}
	}()
	return decodeBody[T](response.Body, link.Rel)
}

func decodeBody[T any](body io.Reader, rel string) (T, error) {
	var result T
	if err := json.NewDecoder(io.LimitReader(body, maxResponseBytes)).Decode(&result); err != nil {
		return result, fmt.Errorf("studio: decode %q response: %w", rel, err)
	}
	return result, nil
}

func responseError(response *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorBytes))
	closeErr := response.Body.Close()

	apiErr := &APIError{
		Status:          response.StatusCode,
		WWWAuthenticate: response.Header.Get("WWW-Authenticate"),
	}
	var envelope ErrorBody
	if err := json.Unmarshal(body, &envelope); err == nil {
		apiErr.Body = envelope
	}

	if readErr != nil {
		return errors.Join(apiErr, fmt.Errorf("studio: read error response: %w", readErr))
	}
	if closeErr != nil {
		return errors.Join(apiErr, fmt.Errorf("studio: close error response: %w", closeErr))
	}
	return apiErr
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func defaultRetryDelay(attempt int) time.Duration {
	return baseRetryDelay << attempt
}
