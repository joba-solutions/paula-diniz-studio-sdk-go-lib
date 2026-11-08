package studio

import "fmt"

// ErrorBody is the API's error envelope: a summary message and the per-field errors it names.
type ErrorBody struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// APIError is a failed Studio API call, carrying the status so callers branch on it, the parsed
// error envelope, and the edge's www-authenticate reason.
type APIError struct {
	Status          int
	Body            ErrorBody
	WWWAuthenticate string
}

// Error reports the status and the message the server sent.
func (e *APIError) Error() string {
	if e.Body.Message == "" {
		return fmt.Sprintf("studio: HTTP %d", e.Status)
	}
	return fmt.Sprintf("studio: HTTP %d: %s", e.Status, e.Body.Message)
}
