// Package studio is a client for the Paula Diniz Studio API.
//
// The API is hypermedia-driven. A caller starts at HomeLink, resolves the home document into the
// entry points the API advertises, and follows those links. It never builds a path from a string.
package studio
